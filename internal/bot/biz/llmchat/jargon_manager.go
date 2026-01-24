package llmchat

import (
	"LanMei/internal/bot/biz/dao"
	"context"
	"fmt"
	"strings"
	"time"
)

const jargonPrefix = "LanMei-Jargon"

var jargonThresholds = []int64{2, 4, 7, 10, 13, 16, 19, 21}

type JargonManager struct {
	learner    *JargonLearner
	thresholds []int64
}

func NewJargonManager(learner *JargonLearner) *JargonManager {
	return &JargonManager{
		learner:    learner,
		thresholds: jargonThresholds,
	}
}

func (m *JargonManager) ObserveAndExplain(ctx context.Context, groupID, userID string, terms []string, contextText string) string {
	if m == nil || dao.DBManager == nil || len(terms) == 0 {
		return ""
	}
	notes := make([]string, 0, len(terms))
	for _, term := range terms {
		term = strings.TrimSpace(term)
		if term == "" {
			continue
		}
		count, err := dao.DBManager.IncrJargonCount(ctx, groupID, term)
		if err != nil {
			continue
		}
		explanation := m.searchExplanation(ctx, groupID, term)
		if explanation != "" {
			notes = append(notes, explanation)
		}
		m.storeOccurrence(ctx, groupID, userID, term, contextText)
		if m.shouldLearn(ctx, groupID, term, count) {
			contexts := m.collectContexts(ctx, groupID, term)
			m.asyncLearn(groupID, term, contexts, contextText, count)
		}
	}
	return strings.Join(dedupeStrings(notes), "\n")
}

func (m *JargonManager) shouldLearn(ctx context.Context, groupID, term string, count int64) bool {
	if len(m.thresholds) == 0 {
		return false
	}
	lastInfer, err := dao.DBManager.GetJargonLastInfer(ctx, groupID, term)
	if err != nil {
		return false
	}
	if count <= lastInfer {
		return false
	}
	for _, threshold := range m.thresholds {
		if threshold > lastInfer {
			return count >= threshold
		}
	}
	return false
}

func (m *JargonManager) asyncLearn(groupID, term string, contexts []string, contextText string, count int64) {
	if m.learner == nil {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		meaning, ok := m.learner.Infer(ctx, term, contexts, contextText)
		if !ok || meaning.NoInfo || meaning.Meaning == "" {
			return
		}
		m.storeExplanation(ctx, groupID, term, meaning.Meaning)
		_ = dao.DBManager.SetJargonLastInfer(ctx, groupID, term, count)
	}()
}

func (m *JargonManager) searchExplanation(ctx context.Context, groupID, term string) string {
	results := dao.DBManager.GetTopK(ctx, jargonCollection(groupID), 5, fmt.Sprintf("俚语解释:%s", term))
	for _, res := range results {
		if !strings.HasPrefix(res, "俚语解释:") {
			continue
		}
		if parsed := parseJargonExplanation(res); parsed != "" {
			return parsed
		}
	}
	return ""
}

func (m *JargonManager) collectContexts(ctx context.Context, groupID, term string) []string {
	results := dao.DBManager.GetTopK(ctx, jargonCollection(groupID), 6, fmt.Sprintf("俚语上下文:%s", term))
	out := make([]string, 0, len(results))
	for _, res := range results {
		contextText := parseJargonContext(res)
		if contextText == "" {
			continue
		}
		out = append(out, contextText)
	}
	return dedupeStrings(out)
}

func parseJargonExplanation(text string) string {
	trimmed := strings.TrimSpace(strings.TrimPrefix(text, "俚语解释:"))
	if trimmed == "" {
		return ""
	}
	if strings.Contains(trimmed, "->") {
		parts := strings.SplitN(trimmed, "->", 2)
		if len(parts) != 2 {
			return ""
		}
		term := strings.TrimSpace(parts[0])
		meaning := strings.TrimSpace(parts[1])
		if term == "" || meaning == "" {
			return ""
		}
		return fmt.Sprintf("%s: %s", term, meaning)
	}
	return trimmed
}

func parseJargonContext(text string) string {
	trimmed := strings.TrimSpace(strings.TrimPrefix(text, "俚语上下文:"))
	if trimmed == "" {
		return ""
	}
	if strings.Contains(trimmed, "->") {
		parts := strings.SplitN(trimmed, "->", 2)
		if len(parts) != 2 {
			return ""
		}
		contextText := strings.TrimSpace(parts[1])
		return contextText
	}
	return trimmed
}

func (m *JargonManager) storeOccurrence(ctx context.Context, groupID, userID, term, contextText string) {
	text := fmt.Sprintf("俚语出现:%s 用户:%s 上下文:%s", term, userID, contextText)
	item := dao.EmbeddingItem{
		ID:   dao.NextEmbeddingID(),
		Text: text,
	}
	_ = dao.DBManager.UpsertEmbeddingTexts(ctx, jargonCollection(groupID), []dao.EmbeddingItem{item})
	contextItem := dao.EmbeddingItem{
		ID:   dao.NextEmbeddingID(),
		Text: fmt.Sprintf("俚语上下文:%s -> %s", term, contextText),
	}
	_ = dao.DBManager.UpsertEmbeddingTexts(ctx, jargonCollection(groupID), []dao.EmbeddingItem{contextItem})
}

func (m *JargonManager) storeExplanation(ctx context.Context, groupID, term, meaning string) {
	text := fmt.Sprintf("俚语解释:%s -> %s", term, meaning)
	item := dao.EmbeddingItem{
		ID:   dao.NextEmbeddingID(),
		Text: text,
	}
	_ = dao.DBManager.UpsertEmbeddingTexts(ctx, jargonCollection(groupID), []dao.EmbeddingItem{item})
}

func jargonCollection(groupID string) string {
	return fmt.Sprintf("%s-%s", jargonPrefix, groupID)
}
