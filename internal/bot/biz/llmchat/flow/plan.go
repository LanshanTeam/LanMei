package flow

import (
	"context"
	"encoding/json"
	"strings"

	"LanMei/internal/bot/biz/llmchat/hooks"
	"LanMei/internal/bot/utils/llog"

	"github.com/cloudwego/eino/schema"
)

func buildPlan(ctx context.Context, deps Dependencies, state *State) Plan {
	if state == nil || deps.PlanTemplate == nil || deps.PlannerModel == nil {
		return Plan{}
	}
	message := strings.TrimSpace(state.Analysis.OptimizedInput)
	if message == "" {
		message = strings.TrimSpace(state.Analysis.RawInput)
	}
	in, err := deps.PlanTemplate.Format(ctx, map[string]any{
		"nickname":         state.Request.Nickname,
		"history":          state.History,
		"message":          message,
		"intent":           state.Analysis.Intent,
		"purpose":          state.Analysis.Purpose,
		"psych_state":      state.Analysis.PsychState,
		"addressed_target": state.Analysis.AddressedTarget,
		"target_detail":    state.Analysis.TargetDetail,
		"optimized_input":  state.Analysis.OptimizedInput,
	})
	if err != nil {
		llog.Error("format plan message error: %v", err)
		return Plan{}
	}
	msg, err := hooks.Run(ctx, deps.Hooks, deps.HookInfos.Plan, func() (*schema.Message, error) {
		return deps.PlannerModel.Generate(ctx, in)
	})
	if err != nil {
		llog.Error("generate plan error: %v", err)
		return Plan{}
	}
	for _, tc := range msg.ToolCalls {
		if tc.Function.Name != "plan_chat" {
			continue
		}
		var plan Plan
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &plan); err != nil {
			llog.Error("解析 planner tool 参数失败: %v", err)
			break
		}
		llog.Info("plan: ", plan)
		plan.Intent = strings.TrimSpace(plan.Intent)
		plan.Action = strings.TrimSpace(plan.Action)
		plan.ReplyStyle = strings.TrimSpace(plan.ReplyStyle)
		return plan
	}
	return Plan{}
}
