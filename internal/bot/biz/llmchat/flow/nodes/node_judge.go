package nodes

import (
	"context"
	"encoding/json"
	"fmt"

	"LanMei/internal/bot/biz/llmchat/flow/hooks"
	flowtypes "LanMei/internal/bot/biz/llmchat/flow/types"
	"LanMei/internal/bot/utils/llog"

	"github.com/cloudwego/eino/schema"
)

func JudgeNode(deps flowtypes.Dependencies) func(context.Context, *flowtypes.State) (*flowtypes.State, error) {
	return func(ctx context.Context, state *flowtypes.State) (*flowtypes.State, error) {
		if state == nil || state.Stop {
			return state, nil
		}
		if state.Request.Must {
			return state, nil
		}
		if deps.JudgeModel == nil || deps.JudgeTemplate == nil {
			state.StopWith("judge_unavailable")
			return state, nil
		}
		recentAssistantReplies := recentAssistantReplies(state.History, replyFrequencyWindow)
		judgeIn, err := deps.JudgeTemplate.Format(ctx, map[string]any{
			"message":                  state.Request.Input,
			"history":                  state.History,
			"intent":                   state.Analysis.Intent,
			"purpose":                  state.Analysis.Purpose,
			"psych_state":              state.Analysis.PsychState,
			"addressed_target":         state.Analysis.AddressedTarget,
			"target_detail":            state.Analysis.TargetDetail,
			"optimized_input":          state.Analysis.OptimizedInput,
			"recent_assistant_replies": recentAssistantReplies,
			"reply_window":             replyFrequencyWindow,
		})
		if err != nil {
			llog.Error("format judge message error: %v", err)
			state.StopWith("judge_format_error")
			return state, nil
		}
		judgeMsg, err := hooks.Run(ctx, deps.Hooks, deps.HookInfos.Judge, func() (*schema.Message, error) {
			return deps.JudgeModel.Generate(ctx, judgeIn)
		})
		if err != nil {
			llog.Error("generate judge message error: %v", err)
			state.StopWith("judge_error")
			return state, nil
		}
		if len(judgeMsg.ToolCalls) == 0 {
			state.StopWith("judge_no_tool")
			return state, nil
		}
		for _, tc := range judgeMsg.ToolCalls {
			if tc.Function.Name != "interested_scores" {
				continue
			}
			var params map[string]interface{}
			if err := json.Unmarshal([]byte(tc.Function.Arguments), &params); err != nil {
				llog.Error("unmarshal arguments error: %v", err)
				state.StopWith("judge_unmarshal_error")
				return state, nil
			}
			score, ok := computeReplyScore(params)
			if !ok {
				state.StopWith("judge_score_blocked")
				return state, nil
			}
			repeatPenalty := clampPenalty(toFloat(params["repeat_penalty"]))
			frequencyPenalty := clampPenalty(toFloat(params["frequency_penalty"]))
			penalty := repeatPenalty + frequencyPenalty
			if penalty > replyPenaltyMax {
				penalty = replyPenaltyMax
			}
			threshold := baseReplyScoreThreshold + penalty
			llog.Info(fmt.Sprintf("should Reply: params=%v score=%.1f penalty=%.1f threshold=%.1f", params, score, penalty, threshold))
			if score >= threshold {
				return state, nil
			}
			state.StopWith("judge_threshold_blocked")
			return state, nil
		}
		state.StopWith("judge_no_score")
		return state, nil
	}
}
