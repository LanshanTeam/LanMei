package nodes

import (
	"context"

	flowtypes "LanMei/internal/bot/biz/llmchat/flow/types"
	"LanMei/internal/bot/utils/sensitive"
)

func PostProcessNode(deps flowtypes.Dependencies) func(context.Context, *flowtypes.State) (*flowtypes.State, error) {
	return func(ctx context.Context, state *flowtypes.State) (*flowtypes.State, error) {
		if state == nil || state.Stop {
			return state, nil
		}
		if sensitive.HaveSensitive(state.Reply) {
			state.Reply = "唔唔~小蓝的数据库里没有这种词哦，要不要换个萌萌的说法呀~(>ω<)"
			state.StopWith("sensitive_blocked")
			return state, nil
		}
		if deps.Memory != nil {
			deps.Memory.AppendAssistant(state.Request.GroupID, state.Reply)
		}
		if deps.Frequency != nil {
			deps.Frequency.MarkSent(state.Request.GroupID)
		}
		return state, nil
	}
}
