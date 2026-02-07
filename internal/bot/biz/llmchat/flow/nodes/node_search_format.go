package nodes

import (
	"context"
	"strings"

	"LanMei/internal/bot/biz/llmchat/flow/hooks"
	flowtypes "LanMei/internal/bot/biz/llmchat/flow/types"
	"LanMei/internal/bot/utils/llog"

	"github.com/cloudwego/eino/schema"
)

func SearchFormatNode(deps flowtypes.Dependencies) func(context.Context, *flowtypes.State) (*flowtypes.State, error) {
	return func(ctx context.Context, state *flowtypes.State) (*flowtypes.State, error) {
		if state == nil || state.Stop {
			return state, nil
		}
		state.WebSearch = formatSearchResults(ctx, deps, state.Analysis.RawInput, state.Analysis.SearchQueries, state.WebSearchRaw)
		return state, nil
	}
}

func formatSearchResults(ctx context.Context, deps flowtypes.Dependencies, input string, queries []string, raw string) string {
	if deps.SearchModel == nil || deps.SearchTemplate == nil {
		return raw
	}
	if strings.TrimSpace(raw) == "" || strings.TrimSpace(raw) == "无" {
		return raw
	}
	in, err := deps.SearchTemplate.Format(ctx, map[string]any{
		"input":       input,
		"queries":     strings.Join(queries, "、"),
		"raw_results": raw,
	})
	if err != nil {
		llog.Error("format search summary error: %v", err)
		return raw
	}
	msg, err := hooks.Run(ctx, deps.Hooks, deps.HookInfos.Search, func() (*schema.Message, error) {
		return deps.SearchModel.Generate(ctx, in)
	})
	if err != nil {
		llog.Error("generate search summary error: %v", err)
		return raw
	}
	content := strings.TrimSpace(msg.Content)
	if content == "" {
		return raw
	}
	llog.Info("格式化后的结果", content)
	return content
}
