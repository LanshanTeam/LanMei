package nodes

import (
	"context"
	"strings"
	"time"

	flowtypes "LanMei/internal/bot/biz/llmchat/flow/types"
	"LanMei/internal/bot/utils/llog"
)

func BuildPromptNode(deps flowtypes.Dependencies) func(context.Context, *flowtypes.State) (*flowtypes.State, error) {
	return func(ctx context.Context, state *flowtypes.State) (*flowtypes.State, error) {
		if state == nil || state.Stop {
			return state, nil
		}
		if deps.Template == nil {
			state.StopWith("template_missing")
			return state, nil
		}
		rawInput := strings.TrimSpace(state.Analysis.RawInput)
		if rawInput == "" {
			rawInput = state.Analysis.OptimizedInput
		}
		augmentedInput := state.Request.Nickname + "说：" + rawInput
		promptInput, err := deps.Template.Format(ctx, map[string]any{
			"message":          augmentedInput,
			"time":             time.Now(),
			"feishu":           state.Knowledge,
			"history":          state.History,
			"memory":           state.MemoryBlock,
			"web_search":       state.WebSearch,
			"user_profile":     formatUserProfile(state.UserProfile),
			"user_facts":       formatUserFacts(state.UserFacts),
			"plan":             formatPlan(state.Plan),
			"intent":           state.Analysis.Intent,
			"purpose":          state.Analysis.Purpose,
			"psych_state":      state.Analysis.PsychState,
			"addressed_target": state.Analysis.AddressedTarget,
			"target_detail":    state.Analysis.TargetDetail,
			"raw_input":        rawInput,
			"optimized_input":  state.Analysis.OptimizedInput,
			"reply_style":      state.Plan.ReplyStyle,
		})
		if err != nil {
			llog.Error("format message error: %v", err)
			state.Reply = state.Request.Input
			state.StopWith("prompt_format_error")
			return state, nil
		}
		state.Prompt = promptInput
		return state, nil
	}
}

func formatUserFacts(facts []string) string {
	if len(facts) == 0 {
		return "无"
	}
	return strings.Join(facts, "\n")
}

func formatUserProfile(profile string) string {
	profile = strings.TrimSpace(profile)
	if profile == "" {
		return "无"
	}
	return profile
}
