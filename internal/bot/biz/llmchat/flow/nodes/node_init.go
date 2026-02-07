package nodes

import (
	"context"

	flowtypes "LanMei/internal/bot/biz/llmchat/flow/types"
	"LanMei/internal/bot/biz/llmchat/memory"
	"LanMei/internal/bot/utils/llog"

	"github.com/cloudwego/eino/schema"
)

func InitNode(deps flowtypes.Dependencies) func(context.Context, *flowtypes.State) (*flowtypes.State, error) {
	return func(ctx context.Context, state *flowtypes.State) (*flowtypes.State, error) {
		if state == nil {
			return state, nil
		}
		state.History = loadAndStoreHistory(deps.Memory, state.Request.GroupID, state.Request.UserID, state.Request.Nickname, state.Request.Input)
		if !state.Request.Must && deps.Frequency != nil && deps.Frequency.ShouldThrottle(state.Request.GroupID) {
			llog.Info("频率控制，不回复")
			state.StopWith("throttle")
			return state, nil
		}
		return state, nil
	}
}

func loadAndStoreHistory(memoryManager *memory.MemoryManager, groupID, userID, nickname, input string) []schema.Message {
	if memoryManager == nil {
		return []schema.Message{}
	}
	return memoryManager.LoadHistoryAndAppendUser(groupID, userID, nickname, input)
}
