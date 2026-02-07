package nodes

import (
	"context"
	"strings"

	flowtypes "LanMei/internal/bot/biz/llmchat/flow/types"
)

func UserContextNode(deps flowtypes.Dependencies) func(context.Context, *flowtypes.State) (*flowtypes.State, error) {
	return func(ctx context.Context, state *flowtypes.State) (*flowtypes.State, error) {
		if state == nil || state.Stop {
			return state, nil
		}
		if deps.Memory == nil {
			state.UserFacts = nil
			state.UserProfile = "无"
			return state, nil
		}
		qqid := strings.TrimSpace(state.Request.UserID)
		if qqid == "" {
			qqid = strings.TrimSpace(state.Request.Nickname)
		}
		facts, profile := deps.Memory.GetUserContext(ctx, state.Request.GroupID, qqid, 12)
		state.UserFacts = facts
		state.UserProfile = profile
		return state, nil
	}
}
