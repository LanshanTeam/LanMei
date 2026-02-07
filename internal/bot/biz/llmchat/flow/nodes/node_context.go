package nodes

import (
	"context"
	"strings"

	"LanMei/internal/bot/biz/dao"
	flowtypes "LanMei/internal/bot/biz/llmchat/flow/types"
	"LanMei/internal/bot/biz/llmchat/memory"
)

func GatherContextNode(deps flowtypes.Dependencies) func(context.Context, *flowtypes.State) (*flowtypes.State, error) {
	return func(ctx context.Context, state *flowtypes.State) (*flowtypes.State, error) {
		if state == nil || state.Stop {
			return state, nil
		}
		state.MemoryBlock = recallMemory(ctx, deps.Memory, state.Analysis.OptimizedInput, state.Request.GroupID, state.Plan.NeedMemory)
		state.Knowledge = recallKnowledge(ctx, deps, state.Analysis.OptimizedInput, state.Plan.NeedKnowledge)
		state.WebSearchRaw = recallWebSearchRaw(ctx, deps, state.Analysis)
		return state, nil
	}
}

func recallMemory(ctx context.Context, memoryManager *memory.MemoryManager, query, groupID string, needMemory bool) string {
	if memoryManager == nil || !needMemory {
		return "无"
	}
	memorySnippets := memoryManager.Retrieve(ctx, query, groupID, needMemory)
	if len(memorySnippets) == 0 {
		return "无"
	}
	return strings.Join(memorySnippets, "\n")
}

func recallKnowledge(ctx context.Context, deps flowtypes.Dependencies, query string, needKnowledge bool) []string {
	if !needKnowledge {
		return nil
	}
	if dao.DBManager == nil {
		return nil
	}
	msgs := dao.DBManager.GetTopK(ctx, dao.CollectionName, 50, query)
	if deps.Reranker != nil {
		reranked := deps.Reranker.TopN(8, msgs, query)
		if len(reranked) > 0 {
			msgs = reranked
		}
	}
	return msgs
}
