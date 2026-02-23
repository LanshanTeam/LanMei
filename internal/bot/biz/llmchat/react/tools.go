package react

import (
	"context"
	"fmt"
	"strings"

	"LanMei/internal/bot/biz/dao"
	"LanMei/internal/bot/biz/llmchat/memory"
	"LanMei/internal/bot/biz/llmchat/reactlog"
	"LanMei/internal/bot/utils/llog"
	"LanMei/internal/bot/utils/rerank"
	"LanMei/internal/bot/utils/websearch"

	"github.com/bytedance/sonic"
	"github.com/cloudwego/eino/components/tool"
	toolutils "github.com/cloudwego/eino/components/tool/utils"
)

const (
	toolWebSearch       = "web_search"
	toolRecallMemory    = "recall_memory"
	toolRecallKnowledge = "recall_knowledge"
	ToolFinalResponse   = "final_response"
)

type webSearchInput struct {
	Query   string   `json:"query"`
	Queries []string `json:"queries"`
	Limit   int      `json:"limit"`
}

type recallInput struct {
	Query string `json:"query"`
	Limit int    `json:"limit"`
}

type finalResponseInput struct {
	Action  string `json:"action"`
	Content string `json:"content"`
}

func BuildTools(searcher *websearch.Client, memoryManager *memory.MemoryManager, reranker *rerank.Reranker) ([]tool.BaseTool, error) {
	tools := make([]tool.BaseTool, 0, 4)
	webSearchTool, err := toolutils.InferTool[webSearchInput, string](
		toolWebSearch,
		"使用搜索引擎检索最新信息或外部事实",
		func(ctx context.Context, input webSearchInput) (string, error) {
			return runWebSearch(ctx, searcher, input)
		},
		rawStringOutput(),
	)
	if err != nil {
		return nil, err
	}
	tools = append(tools, webSearchTool)

	recallMemoryTool, err := toolutils.InferTool[recallInput, string](
		toolRecallMemory,
		"检索长期记忆片段",
		func(ctx context.Context, input recallInput) (string, error) {
			return runRecallMemory(ctx, memoryManager, input)
		},
		rawStringOutput(),
	)
	if err != nil {
		return nil, err
	}
	tools = append(tools, recallMemoryTool)

	recallKnowledgeTool, err := toolutils.InferTool[recallInput, string](
		toolRecallKnowledge,
		"检索知识库内容",
		func(ctx context.Context, input recallInput) (string, error) {
			return runRecallKnowledge(ctx, reranker, input)
		},
		rawStringOutput(),
	)
	if err != nil {
		return nil, err
	}
	tools = append(tools, recallKnowledgeTool)

	finalTool, err := toolutils.InferTool[finalResponseInput, finalResponseInput](
		ToolFinalResponse,
		"输出最终动作与回复",
		func(ctx context.Context, input finalResponseInput) (finalResponseInput, error) {
			return normalizeFinalResponse(input), nil
		},
	)
	if err != nil {
		return nil, err
	}
	tools = append(tools, finalTool)

	return tools, nil
}

func rawStringOutput() toolutils.Option {
	return toolutils.WithMarshalOutput(func(_ context.Context, output interface{}) (string, error) {
		if output == nil {
			return "", nil
		}
		switch v := output.(type) {
		case string:
			return v, nil
		default:
			return sonic.MarshalString(v)
		}
	})
}

func normalizeFinalResponse(input finalResponseInput) finalResponseInput {
	input.Action = strings.TrimSpace(input.Action)
	input.Content = strings.TrimSpace(input.Content)
	if input.Action == "" {
		input.Action = "reply"
	}
	return input
}

func runWebSearch(ctx context.Context, searcher *websearch.Client, input webSearchInput) (string, error) {
	if searcher == nil {
		return "无", nil
	}
	queries := make([]string, 0, len(input.Queries)+1)
	if strings.TrimSpace(input.Query) != "" {
		queries = append(queries, strings.TrimSpace(input.Query))
	}
	for _, q := range input.Queries {
		q = strings.TrimSpace(q)
		if q != "" {
			queries = append(queries, q)
		}
	}
	if len(queries) == 0 {
		if fallback := strings.TrimSpace(reactlog.QueryFromContext(ctx)); fallback != "" {
			queries = append(queries, fallback)
		}
	}
	if len(queries) == 0 {
		return "无", nil
	}
	if len(queries) > 3 {
		queries = queries[:3]
	}
	limit := input.Limit
	if limit <= 0 {
		limit = 4
	}
	blocks := make([]string, 0, len(queries))
	for _, query := range queries {
		results, err := searcher.Search(ctx, query, limit)
		if err != nil {
			llog.Error("网络检索失败: %v", err)
			continue
		}
		block := formatWebSearch(results)
		if block == "无" {
			continue
		}
		blocks = append(blocks, fmt.Sprintf("查询:%s -> 获取结果为：%s", query, block))
	}
	if len(blocks) == 0 {
		return "无", nil
	}
	return strings.Join(blocks, "\n"), nil
}

func runRecallMemory(ctx context.Context, memoryManager *memory.MemoryManager, input recallInput) (string, error) {
	if memoryManager == nil {
		return "无", nil
	}
	query := strings.TrimSpace(input.Query)
	if query == "" {
		query = strings.TrimSpace(reactlog.QueryFromContext(ctx))
	}
	if query == "" {
		return "无", nil
	}
	groupID := reactlog.GroupIDFromContext(ctx)
	results := memoryManager.Retrieve(ctx, query, groupID, true)
	if len(results) == 0 {
		return "无", nil
	}
	if input.Limit > 0 && len(results) > input.Limit {
		results = results[:input.Limit]
	}
	return strings.Join(results, "\n"), nil
}

func runRecallKnowledge(ctx context.Context, reranker *rerank.Reranker, input recallInput) (string, error) {
	query := strings.TrimSpace(input.Query)
	if query == "" {
		query = strings.TrimSpace(reactlog.QueryFromContext(ctx))
	}
	if query == "" || dao.DBManager == nil {
		return "无", nil
	}
	msgs := dao.DBManager.GetTopK(ctx, dao.CollectionName, 50, query)
	if reranker != nil {
		reranked := reranker.TopN(8, msgs, query)
		if len(reranked) > 0 {
			msgs = reranked
		}
	}
	if input.Limit > 0 && len(msgs) > input.Limit {
		msgs = msgs[:input.Limit]
	}
	if len(msgs) == 0 {
		return "无", nil
	}
	return strings.Join(msgs, "\n"), nil
}

func formatWebSearch(results []websearch.Result) string {
	if len(results) == 0 {
		return "无"
	}
	lines := make([]string, 0, len(results))
	for _, res := range results {
		line := strings.TrimSpace(res.Title)
		if line == "" {
			continue
		}
		snippet := strings.TrimSpace(res.Snippet)
		if snippet != "" {
			line = fmt.Sprintf("%s - %s", line, snippet)
		}
		if res.URL != "" {
			line = fmt.Sprintf("%s (%s)", line, res.URL)
		}
		lines = append(lines, line)
	}
	if len(lines) == 0 {
		return "无"
	}
	return strings.Join(lines, "\n")
}
