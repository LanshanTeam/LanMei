package nodes

import (
	"context"

	flowtypes "LanMei/internal/bot/biz/llmchat/flow/types"
	"LanMei/internal/bot/utils/llog"
)

func AnalysisNode(deps flowtypes.Dependencies) func(context.Context, *flowtypes.State) (*flowtypes.State, error) {
	return func(ctx context.Context, state *flowtypes.State) (*flowtypes.State, error) {
		if state == nil || state.Stop {
			return state, nil
		}
		if deps.InputAnalyzer == nil {
			state.StopWith("analysis_unavailable")
			return state, nil
		}
		analysisResult, ok := deps.InputAnalyzer.Analyze(ctx, state.Request.Nickname, state.Request.Input, state.History, state.UserFacts, state.UserProfile)
		if !ok {
			state.StopWith("analysis_failed")
			return state, nil
		}
		analysisResult = Normalize(analysisResult, state.Request.Input)
		llog.Info("意图分析：", analysisResult)
		state.Analysis = analysisResult
		return state, nil
	}
}
