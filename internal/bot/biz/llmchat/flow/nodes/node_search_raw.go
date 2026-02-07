package nodes

import (
	"context"
	"fmt"
	"strings"

	flowtypes "LanMei/internal/bot/biz/llmchat/flow/types"
	"LanMei/internal/bot/utils/llog"
)

func recallWebSearchRaw(ctx context.Context, deps flowtypes.Dependencies, input flowtypes.InputAnalysis) string {
	if deps.Searcher == nil || !input.NeedSearch {
		return "无"
	}
	queries := append([]string(nil), input.SearchQueries...)
	if len(queries) == 0 && strings.TrimSpace(input.OptimizedInput) != "" {
		queries = []string{strings.TrimSpace(input.OptimizedInput)}
	}
	maxQueries := 3
	if len(queries) > maxQueries {
		queries = queries[:maxQueries]
	}
	blocks := make([]string, 0, len(queries))
	for _, query := range queries {
		results, err := deps.Searcher.Search(ctx, query, 4)
		if err != nil {
			llog.Error("网络检索失败: %v", err)
			continue
		}
		block := formatWebSearch(results)
		if block == "无" {
			continue
		}
		blocks = append(blocks, fmt.Sprintf("查询:%s -> 获取结果为：%s \n", query, block))
	}
	llog.Info("网络搜索结果：", blocks)
	if len(blocks) == 0 {
		return "无"
	}
	return strings.Join(blocks, "\n")
}
