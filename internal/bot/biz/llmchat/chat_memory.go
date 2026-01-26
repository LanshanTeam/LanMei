package llmchat

import (
	"LanMei/internal/bot/biz/dao"
	"LanMei/internal/bot/utils/llog"
	"LanMei/internal/bot/utils/rerank"
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/eino/schema"
)

const (
	chatMemoryPrefix         = "LanMei-ChatMemory"
	defaultShortMemoryWindow = 20
)

type MemoryMessage struct {
	GroupID  string
	UserID   string
	Nickname string
	Role     schema.RoleType
	Content  string
	At       time.Time
}

type ShortTermMemory struct {
	mu     sync.Mutex
	window int
	groups map[string][]MemoryMessage
}

func NewShortTermMemory(window int) *ShortTermMemory {
	if window <= 0 {
		window = defaultShortMemoryWindow
	}
	return &ShortTermMemory{
		window: window,
		groups: make(map[string][]MemoryMessage),
	}
}

func (s *ShortTermMemory) Snapshot(groupID string) []schema.Message {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	items := s.groups[groupID]
	out := make([]schema.Message, len(items))
	for i, item := range items {
		out[i] = schema.Message{Role: item.Role, Content: item.Content}
	}
	return out
}

func (s *ShortTermMemory) Append(groupID string, msg MemoryMessage) []MemoryMessage {
	if s == nil || s.window <= 0 {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	items := append(s.groups[groupID], msg)
	var evicted []MemoryMessage
	if len(items) > s.window {
		over := len(items) - s.window
		evicted = make([]MemoryMessage, over)
		copy(evicted, items[:over])
		items = append([]MemoryMessage(nil), items[over:]...)
	}
	s.groups[groupID] = items
	return evicted
}

func (s *ShortTermMemory) DrainAll() map[string][]MemoryMessage {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string][]MemoryMessage, len(s.groups))
	for groupID, items := range s.groups {
		if len(items) == 0 {
			continue
		}
		copied := make([]MemoryMessage, len(items))
		copy(copied, items)
		out[groupID] = copied
	}
	s.groups = make(map[string][]MemoryMessage)
	return out
}

type MemoryManager struct {
	reranker  *rerank.Reranker
	extractor *MemoryExtractor
	shortTerm *ShortTermMemory
	worker    *MemoryWorker
}

func NewMemoryManager(reranker *rerank.Reranker, extractor *MemoryExtractor, windowSize int) *MemoryManager {
	return &MemoryManager{
		reranker:  reranker,
		extractor: extractor,
		shortTerm: NewShortTermMemory(windowSize),
	}
}

func (m *MemoryManager) BindWorker(worker *MemoryWorker) {
	if m == nil {
		return
	}
	m.worker = worker
}

func (m *MemoryManager) LoadHistoryAndAppendUser(groupID, userID, nickname, input string) []schema.Message {
	if m == nil || m.shortTerm == nil {
		return []schema.Message{}
	}
	history := m.shortTerm.Snapshot(groupID)
	if strings.TrimSpace(input) == "" {
		return history
	}
	evicted := m.shortTerm.Append(groupID, MemoryMessage{
		GroupID:  groupID,
		UserID:   userID,
		Nickname: nickname,
		Role:     schema.User,
		Content:  input,
		At:       time.Now(),
	})
	m.enqueueEvicted(groupID, evicted, false)
	return history
}

func (m *MemoryManager) AppendAssistant(groupID, reply string) {
	if m == nil || m.shortTerm == nil {
		return
	}
	if strings.TrimSpace(reply) == "" {
		return
	}
	evicted := m.shortTerm.Append(groupID, MemoryMessage{
		GroupID: groupID,
		Role:    schema.Assistant,
		Content: reply,
		At:      time.Now(),
	})
	m.enqueueEvicted(groupID, evicted, false)
}

func (m *MemoryManager) FlushAll() {
	if m == nil || m.shortTerm == nil {
		return
	}
	drained := m.shortTerm.DrainAll()
	for groupID, messages := range drained {
		m.enqueueEvicted(groupID, messages, true)
	}
	if m.worker != nil {
		m.worker.Stop()
	}
}

func (m *MemoryManager) enqueueEvicted(groupID string, messages []MemoryMessage, block bool) {
	if m == nil || m.worker == nil || len(messages) == 0 {
		return
	}
	if block {
		m.worker.EnqueueBlocking(groupID, messages)
		return
	}
	m.worker.Enqueue(groupID, messages)
}

func (m *MemoryManager) ExtractEvent(ctx context.Context, groupID string, messages []MemoryMessage, force bool) MemoryExtraction {
	if len(messages) == 0 {
		return MemoryExtraction{}
	}
	if m != nil && m.extractor != nil {
		extraction := m.extractor.ExtractBatch(ctx, groupID, messages, force)
		if extraction.Sufficient {
			return extraction
		}
		if !force {
			return extraction
		}
		if hasExtractionContent(extraction) {
			extraction.Sufficient = true
			return extraction
		}
	}
	return fallbackExtraction(messages)
}

func (m *MemoryManager) StoreEvent(ctx context.Context, groupID string, extraction MemoryExtraction) {
	if dao.DBManager == nil {
		return
	}
	text := formatMemoryEventText(extraction)
	if text == "" {
		return
	}
	item := dao.EmbeddingItem{
		ID:   dao.NextEmbeddingID(),
		Text: text,
	}
	if err := dao.DBManager.UpsertEmbeddingTexts(ctx, chatMemoryCollection(groupID), []dao.EmbeddingItem{item}); err != nil {
		llog.Error("写入群聊记忆失败: %v", err)
	}
}

func (m *MemoryManager) Retrieve(ctx context.Context, query, groupID string, needMemory bool) []string {
	if !needMemory || query == "" || dao.DBManager == nil {
		return nil
	}
	merged := dao.DBManager.GetTopK(ctx, chatMemoryCollection(groupID), 8, query)
	merged = dedupeStrings(merged)
	if m == nil || m.reranker == nil {
		return merged
	}
	reranked := m.reranker.TopN(6, merged, query)
	if len(reranked) == 0 {
		return merged
	}
	return reranked
}

func chatMemoryCollection(groupID string) string {
	return fmt.Sprintf("%s-%s", chatMemoryPrefix, groupID)
}

func formatMemoryEventText(extraction MemoryExtraction) string {
	participants := dedupeStrings(extraction.Participants)
	participantsText := strings.Join(participants, "、")
	cause := strings.TrimSpace(extraction.Cause)
	process := strings.TrimSpace(extraction.Process)
	result := strings.TrimSpace(extraction.Result)
	if participantsText == "" && cause == "" && process == "" && result == "" {
		return ""
	}
	if participantsText == "" {
		participantsText = "未知"
	}
	if cause == "" {
		cause = "无"
	}
	if process == "" {
		process = "无"
	}
	if result == "" {
		result = "无"
	}
	return fmt.Sprintf("群聊记忆: 参与者:%s 起因:%s 经过:%s 结果:%s", participantsText, cause, process, result)
}

func hasExtractionContent(extraction MemoryExtraction) bool {
	if len(extraction.Participants) > 0 {
		return true
	}
	if strings.TrimSpace(extraction.Cause) != "" {
		return true
	}
	if strings.TrimSpace(extraction.Process) != "" {
		return true
	}
	if strings.TrimSpace(extraction.Result) != "" {
		return true
	}
	return false
}

func fallbackExtraction(messages []MemoryMessage) MemoryExtraction {
	participants := collectParticipants(messages)
	cause := ""
	for _, msg := range messages {
		if msg.Role == schema.User {
			cause = strings.TrimSpace(msg.Content)
			if cause != "" {
				break
			}
		}
	}
	process := summarizeMessages(messages, 6)
	result := ""
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == schema.Assistant {
			result = strings.TrimSpace(messages[i].Content)
			if result != "" {
				break
			}
		}
	}
	if result == "" && len(messages) > 0 {
		result = strings.TrimSpace(messages[len(messages)-1].Content)
	}
	return MemoryExtraction{
		Sufficient:   true,
		Participants: participants,
		Cause:        truncateText(cause, 80),
		Process:      truncateText(process, 120),
		Result:       truncateText(result, 80),
	}
}

func collectParticipants(messages []MemoryMessage) []string {
	if len(messages) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(messages))
	out := make([]string, 0, len(messages))
	for _, msg := range messages {
		name := memoryMessageSpeaker(msg)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	return out
}

func summarizeMessages(messages []MemoryMessage, limit int) string {
	if len(messages) == 0 {
		return ""
	}
	if limit <= 0 || limit > len(messages) {
		limit = len(messages)
	}
	var builder strings.Builder
	for i := 0; i < limit; i++ {
		if i > 0 {
			builder.WriteString(" | ")
		}
		speaker := memoryMessageSpeaker(messages[i])
		if speaker == "" {
			speaker = "用户"
		}
		builder.WriteString(speaker)
		builder.WriteString(":")
		builder.WriteString(strings.TrimSpace(messages[i].Content))
	}
	return builder.String()
}

func truncateText(text string, max int) string {
	if max <= 0 {
		return ""
	}
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return ""
	}
	runes := []rune(trimmed)
	if len(runes) <= max {
		return trimmed
	}
	return string(runes[:max]) + "..."
}

func memoryMessageSpeaker(msg MemoryMessage) string {
	if msg.Role == schema.Assistant {
		return "蓝妹"
	}
	if msg.Nickname != "" && msg.UserID != "" {
		return fmt.Sprintf("%s(%s)", msg.Nickname, msg.UserID)
	}
	if msg.Nickname != "" {
		return msg.Nickname
	}
	return msg.UserID
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
