package llmchat

import (
	"LanMei/internal/bot/biz/dao"
	"LanMei/internal/bot/utils/llog"
	"LanMei/internal/bot/utils/rerank"
	"context"
	"fmt"
)

const (
	chatMemoryPrefix = "LanMei-ChatMemory"
)

type MemoryManager struct {
	reranker  *rerank.Reranker
	extractor *MemoryExtractor
}

func NewMemoryManager(reranker *rerank.Reranker, extractor *MemoryExtractor) *MemoryManager {
	return &MemoryManager{
		reranker:  reranker,
		extractor: extractor,
	}
}

func chatMemoryCollection(groupID string) string {
	return fmt.Sprintf("%s-%s", chatMemoryPrefix, groupID)
}

func (m *MemoryManager) StoreBatch(ctx context.Context, groupID string, events []MemoryEvent) {
	if len(events) == 0 || dao.DBManager == nil {
		return
	}
	extraction := MemoryExtraction{}
	if m.extractor != nil {
		extraction = m.extractor.ExtractBatch(ctx, groupID, events)
	}
	if extraction.Summary == "" && len(extraction.Facts) == 0 {
		return
	}
	chatItems := make([]dao.EmbeddingItem, 0, 1+len(extraction.Facts))
	if extraction.Summary != "" {
		chatItems = append(chatItems, dao.EmbeddingItem{
			ID:   dao.NextEmbeddingID(),
			Text: fmt.Sprintf("群聊记忆: %s", extraction.Summary),
		})
	}
	for _, fact := range extraction.Facts {
		chatItems = append(chatItems, dao.EmbeddingItem{
			ID:   dao.NextEmbeddingID(),
			Text: "群聊事实: " + fact,
		})
	}
	if err := dao.DBManager.UpsertEmbeddingTexts(ctx, chatMemoryCollection(groupID), chatItems); err != nil {
		llog.Error("写入群聊记忆失败: %v", err)
	}
}

func (m *MemoryManager) Retrieve(ctx context.Context, query, groupID string, needMemory bool) []string {
	if !needMemory || query == "" || dao.DBManager == nil {
		return nil
	}
	merged := dao.DBManager.GetTopK(ctx, chatMemoryCollection(groupID), 8, query)
	merged = dedupeStrings(merged)
	if m.reranker == nil {
		return merged
	}
	reranked := m.reranker.TopN(6, merged, query)
	if len(reranked) == 0 {
		return merged
	}
	return reranked
}

func dedupeStrings(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	out := make([]string, 0, len(items))
	for _, item := range items {
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}
