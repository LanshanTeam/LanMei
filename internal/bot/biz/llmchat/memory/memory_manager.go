package memory

import (
	"LanMei/internal/bot/biz/dao"
	"LanMei/internal/bot/biz/model"
	"LanMei/internal/bot/utils/llog"
	"LanMei/internal/bot/utils/rerank"
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/eino/schema"
)

const (
	factMemoryPrefix         = "LanMei-UserFacts"
	defaultShortMemoryWindow = 20
	minFactConfidence        = 0.45
	defaultFactTopK          = 6
	defaultUserFactTopK      = 12
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
	reranker       *rerank.Reranker
	extractor      *FactExtractor
	updater        *FactUpdater
	profileUpdater *ProfileUpdater
	shortTerm      *ShortTermMemory
	worker         *MemoryWorker
	knownUsersMu   sync.Mutex
	knownUsers     map[string]knownUser
	userAliases    map[string]string
}

func NewMemoryManager(reranker *rerank.Reranker, extractor *FactExtractor, updater *FactUpdater, profileUpdater *ProfileUpdater, windowSize int) *MemoryManager {
	return &MemoryManager{
		reranker:       reranker,
		extractor:      extractor,
		updater:        updater,
		profileUpdater: profileUpdater,
		shortTerm:      NewShortTermMemory(windowSize),
		knownUsers:     make(map[string]knownUser),
		userAliases:    make(map[string]string),
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
	m.registerUser(groupID, userID, nickname)
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

func (m *MemoryManager) ExtractFacts(ctx context.Context, groupID string, messages []MemoryMessage, force bool) FactExtraction {
	if len(messages) == 0 {
		return FactExtraction{}
	}
	if m == nil || m.extractor == nil {
		return FactExtraction{}
	}
	extraction := m.extractor.ExtractBatch(ctx, groupID, messages, force)
	if extraction.Sufficient {
		return extraction
	}
	if !force {
		return extraction
	}
	if len(extraction.Facts) > 0 {
		extraction.Sufficient = true
		return extraction
	}
	return FactExtraction{}
}

func (m *MemoryManager) ApplyFacts(ctx context.Context, groupID string, facts []Fact, force bool) {
	if m == nil || len(facts) == 0 || dao.DBManager == nil {
		return
	}
	changedSubjects := make(map[string]struct{})
	eventFacts := make(map[string][]string)
	// 遍历提取的事实，过滤掉置信度较低的，并准备更新长期记忆
	for _, fact := range facts {
		if !force && fact.Confidence > 0 && fact.Confidence < minFactConfidence {
			continue
		}
		subject := strings.TrimSpace(fact.Subject)
		content := strings.TrimSpace(fact.Content)
		if subject == "" || content == "" {
			continue
		}
		canonical := m.resolveSubject(groupID, subject)
		if canonical == "" {
			continue
		}
		eventFacts[canonical] = append(eventFacts[canonical], formatUserFact(canonical, content))
		// 搜索长期记忆中与该事实相关的旧事实，判断是新增、更新还是删除
		oldFacts := m.searchSimilarFacts(ctx, groupID, canonical, content, defaultFactTopK)
		if m.updater == nil || len(oldFacts) == 0 {
			m.addFact(ctx, groupID, canonical, content)
			if m.isKnownUser(groupID, canonical) {
				changedSubjects[canonical] = struct{}{}
			}
			continue
		}
		// 通过 LLM 决定更新策略
		decision := m.updater.Decide(ctx, canonical, content, oldFacts)
		// 根据决策结果应用到长期记忆中，并记录有哪些主体的事实发生了变化
		if m.applyDecision(ctx, groupID, canonical, content, decision, oldFacts) {
			if m.isKnownUser(groupID, canonical) {
				changedSubjects[canonical] = struct{}{}
			}
		}
	}
	// 这里是需要根据刚更新的事实来更新用户画像。
	if len(changedSubjects) > 0 {
		m.updateProfilesForSubjects(ctx, groupID, changedSubjects, eventFacts)
	}
}

func (m *MemoryManager) Retrieve(ctx context.Context, query, groupID string, needMemory bool) []string {
	if !needMemory || query == "" || dao.DBManager == nil {
		return nil
	}
	merged := dao.DBManager.GetTopK(ctx, factCollection(groupID), 8, query)
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

func (m *MemoryManager) GetUserContext(ctx context.Context, groupID, qqid string, limit int) ([]string, string) {
	facts := m.GetUserFacts(ctx, groupID, qqid, limit)
	profile := m.GetUserProfile(ctx, groupID, qqid)
	return facts, profile
}

func (m *MemoryManager) GetUserFacts(ctx context.Context, groupID, qqid string, limit int) []string {
	qqid = strings.TrimSpace(qqid)
	if qqid == "" || dao.DBManager == nil {
		return nil
	}
	if limit <= 0 {
		limit = defaultUserFactTopK
	}
	query := "用户:" + qqid
	results := dao.DBManager.SearchTopKResults(ctx, factCollection(groupID), query, uint64(limit))
	facts := make([]string, 0, len(results))
	for _, res := range results {
		text := extractFactText(res.Text)
		subject := extractFactSubject(res.Text)
		if subject != "" && subject != qqid {
			continue
		}
		if subject == "" && !strings.Contains(res.Text, qqid) {
			continue
		}
		formatted := formatUserFact(qqid, text)
		if formatted == "" {
			continue
		}
		facts = append(facts, formatted)
	}
	facts = dedupeStrings(facts)
	if m != nil && m.reranker != nil && len(facts) > 1 {
		reranked := m.reranker.TopN(8, facts, qqid)
		if len(reranked) > 0 {
			facts = reranked
		}
	}
	return facts
}

func (m *MemoryManager) GetUserProfile(ctx context.Context, groupID, qqid string) string {
	qqid = strings.TrimSpace(qqid)
	if qqid == "" || dao.DBManager == nil {
		return "无"
	}
	profile, err := dao.DBManager.GetUserProfile(ctx, groupID, qqid)
	if err != nil || profile == nil {
		return "无"
	}
	if strings.TrimSpace(profile.Summary) == "" {
		return "无"
	}
	return profile.Summary
}

func (m *MemoryManager) UpdateProfiles(ctx context.Context) {
	if m == nil || m.profileUpdater == nil || dao.DBManager == nil {
		return
	}
	users := m.listKnownUsers()
	subjects := make(map[string]struct{}, len(users))
	for _, user := range users {
		if user.QQID == "" {
			continue
		}
		subjects[user.QQID] = struct{}{}
	}
	m.updateProfilesForSubjects(ctx, "", subjects, nil)
}

func (m *MemoryManager) updateProfilesForSubjects(ctx context.Context, groupID string, subjects map[string]struct{}, eventFacts map[string][]string) {
	if m == nil || m.profileUpdater == nil || dao.DBManager == nil || len(subjects) == 0 {
		return
	}
	// 遍历用户对象，逐个更新
	for subject := range subjects {
		m.updateProfileForSubject(ctx, groupID, subject, eventFacts[subject])
	}
}

func (m *MemoryManager) updateProfileForSubject(ctx context.Context, groupID, qqid string, extraFacts []string) {
	qqid = strings.TrimSpace(qqid)
	if qqid == "" {
		return
	}
	if groupID == "" {
		for _, user := range m.listKnownUsers() {
			if user.QQID != qqid {
				continue
			}
			m.updateProfileForSubject(ctx, user.GroupID, qqid, extraFacts)
		}
		return
	}
	// 获取和用户相关的事实
	facts := m.GetUserFacts(ctx, groupID, qqid, defaultUserFactTopK)
	facts = mergeEventFacts(extraFacts, facts)
	if len(facts) == 0 {
		return
	}
	current := ProfileResult{}
	existing, err := dao.DBManager.GetUserProfile(ctx, groupID, qqid)
	if err == nil && existing != nil {
		current.Summary = existing.Summary
		current.Tags = splitTags(existing.Tags)
	}
	// 这里也是基于 LLM 进行更新
	updated := m.profileUpdater.Update(ctx, qqid, facts, current)
	if strings.TrimSpace(updated.Summary) == "" {
		return
	}
	nickname := m.lookupNickname(groupID, qqid)
	profile := &model.UserProfile{
		GroupID:  groupID,
		QQID:     qqid,
		Nickname: nickname,
		Summary:  updated.Summary,
		Tags:     strings.Join(updated.Tags, ","),
	}
	if err := dao.DBManager.UpsertUserProfile(ctx, profile); err != nil {
		llog.Errorf("更新用户画像失败: %v", err)
	}
}

type factSearchQuery struct {
	query         string
	filterSubject bool
}

func (m *MemoryManager) searchSimilarFacts(ctx context.Context, groupID, subject, content string, limit int) []FactRecord {
	if dao.DBManager == nil {
		return nil
	}
	if limit <= 0 {
		limit = defaultFactTopK
	}
	queries := buildFactSearchQueries(subject, content)
	if len(queries) == 0 {
		return nil
	}
	perQuery := limit * 2
	seen := make(map[uint64]struct{}, limit*2)
	out := make([]FactRecord, 0, limit)
	for _, q := range queries {
		results := dao.DBManager.SearchTopKResults(ctx, factCollection(groupID), q.query, uint64(perQuery))
		for _, res := range results {
			if len(out) >= limit {
				return out
			}
			if res.ID == 0 {
				continue
			}
			if _, ok := seen[res.ID]; ok {
				continue
			}
			text := strings.TrimSpace(res.Text)
			if text == "" {
				continue
			}
			if q.filterSubject {
				subjectInText := extractFactSubject(text)
				if subjectInText != "" && subjectInText != subject {
					continue
				}
				if subjectInText == "" && !strings.Contains(text, subject) {
					continue
				}
			}
			seen[res.ID] = struct{}{}
			out = append(out, FactRecord{ID: res.ID, Text: text})
		}
	}
	return out
}

func (m *MemoryManager) applyDecision(ctx context.Context, groupID, subject, fallback string, decision FactDecision, oldFacts []FactRecord) bool {
	event := strings.ToUpper(strings.TrimSpace(decision.Event))
	switch event {
	case "UPDATE":
		id := parseDecisionID(decision.TargetID)
		if id == 0 || !hasFactID(oldFacts, id) {
			m.addFact(ctx, groupID, subject, pickDecisionText(decision, fallback))
			return true
		}
		text := normalizeFactText(subject, pickDecisionText(decision, fallback))
		m.upsertFact(ctx, groupID, id, text)
		return true
	case "DELETE":
		id := parseDecisionID(decision.TargetID)
		if id == 0 || !hasFactID(oldFacts, id) {
			return false
		}
		if err := dao.DBManager.DeleteEmbeddingByID(ctx, factCollection(groupID), id); err != nil {
			llog.Errorf("删除事实失败: %v", err)
		}
		return true
	case "NONE":
		return false
	default:
		m.addFact(ctx, groupID, subject, pickDecisionText(decision, fallback))
		return true
	}
}

func (m *MemoryManager) addFact(ctx context.Context, groupID, subject, content string) {
	text := normalizeFactText(subject, content)
	if text == "" {
		return
	}
	id := dao.NextEmbeddingID()
	m.upsertFact(ctx, groupID, id, text)
}

func (m *MemoryManager) upsertFact(ctx context.Context, groupID string, id uint64, text string) {
	if dao.DBManager == nil || id == 0 || strings.TrimSpace(text) == "" {
		return
	}
	item := dao.EmbeddingItem{ID: id, Text: text}
	if err := dao.DBManager.UpsertEmbeddingTexts(ctx, factCollection(groupID), []dao.EmbeddingItem{item}); err != nil {
		llog.Errorf("写入事实失败: %v", err)
	}
}

func (m *MemoryManager) registerUser(groupID, qqid, nickname string) {
	groupID = strings.TrimSpace(groupID)
	qqid = strings.TrimSpace(qqid)
	nickname = strings.TrimSpace(nickname)
	if groupID == "" || qqid == "" {
		return
	}
	key := groupID + "|" + qqid
	m.knownUsersMu.Lock()
	m.knownUsers[key] = knownUser{key: UserKey{GroupID: groupID, QQID: qqid}, Nickname: nickname, lastSeen: time.Now()}
	if nickname != "" {
		aliasKey := groupID + "|" + nickname
		m.userAliases[aliasKey] = qqid
	}
	m.knownUsersMu.Unlock()
}

func (m *MemoryManager) listKnownUsers() []UserKey {
	m.knownUsersMu.Lock()
	defer m.knownUsersMu.Unlock()
	out := make([]UserKey, 0, len(m.knownUsers))
	for _, user := range m.knownUsers {
		out = append(out, user.key)
	}
	return out
}

func (m *MemoryManager) lookupNickname(groupID, qqid string) string {
	if m == nil {
		return ""
	}
	key := strings.TrimSpace(groupID) + "|" + strings.TrimSpace(qqid)
	m.knownUsersMu.Lock()
	user, ok := m.knownUsers[key]
	m.knownUsersMu.Unlock()
	if !ok {
		return ""
	}
	return strings.TrimSpace(user.Nickname)
}

func (m *MemoryManager) resolveSubject(groupID, subject string) string {
	subject = strings.TrimSpace(subject)
	if subject == "" {
		return ""
	}
	key := strings.TrimSpace(groupID) + "|" + subject
	m.knownUsersMu.Lock()
	if _, ok := m.knownUsers[key]; ok {
		m.knownUsersMu.Unlock()
		return subject
	}
	qqid, ok := m.userAliases[key]
	m.knownUsersMu.Unlock()
	if ok && strings.TrimSpace(qqid) != "" {
		return qqid
	}
	return subject
}

func (m *MemoryManager) isKnownUser(groupID, qqid string) bool {
	groupID = strings.TrimSpace(groupID)
	qqid = strings.TrimSpace(qqid)
	if groupID == "" || qqid == "" {
		return false
	}
	key := groupID + "|" + qqid
	m.knownUsersMu.Lock()
	_, ok := m.knownUsers[key]
	m.knownUsersMu.Unlock()
	return ok
}

func factCollection(groupID string) string {
	return fmt.Sprintf("%s-%s", factMemoryPrefix, groupID)
}

func normalizeFactText(subject, content string) string {
	content = strings.TrimSpace(content)
	subject = strings.TrimSpace(subject)
	if content == "" || subject == "" {
		return ""
	}
	if strings.Contains(content, "用户:") || strings.Contains(content, "事实:") {
		return content
	}
	return fmt.Sprintf("用户:%s | 事实:%s", subject, content)
}

func buildFactSearchQueries(subject, content string) []factSearchQuery {
	subject = strings.TrimSpace(subject)
	content = strings.TrimSpace(content)
	if subject == "" && content == "" {
		return nil
	}
	queries := make([]factSearchQuery, 0, 3)
	if subject != "" {
		queries = append(queries, factSearchQuery{
			query:         "用户:" + subject,
			filterSubject: true,
		})
	}
	if content != "" {
		queries = append(queries, factSearchQuery{
			query:         content,
			filterSubject: false,
		})
	}
	return queries
}

func extractFactSubject(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	if !strings.Contains(text, "用户:") {
		return ""
	}
	parts := strings.SplitN(text, "用户:", 2)
	if len(parts) < 2 {
		return ""
	}
	remain := parts[1]
	fields := strings.SplitN(remain, "|", 2)
	subject := strings.TrimSpace(fields[0])
	return subject
}

func extractFactText(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	if !strings.Contains(text, "事实:") {
		return text
	}
	parts := strings.SplitN(text, "事实:", 2)
	if len(parts) < 2 {
		return text
	}
	return strings.TrimSpace(parts[1])
}

func formatUserFact(subject, content string) string {
	content = strings.TrimSpace(content)
	subject = strings.TrimSpace(subject)
	if content == "" {
		return ""
	}
	if subject == "" {
		return content
	}
	display := subject
	if isLikelyQQID(subject) {
		display = "他"
	}
	return display + "：" + content
}

func isLikelyQQID(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func parseDecisionID(raw any) uint64 {
	switch v := raw.(type) {
	case uint64:
		return v
	case int64:
		if v < 0 {
			return 0
		}
		return uint64(v)
	case int:
		if v < 0 {
			return 0
		}
		return uint64(v)
	case float64:
		if v <= 0 {
			return 0
		}
		return uint64(v)
	case string:
		parsed, err := strconv.ParseUint(strings.TrimSpace(v), 10, 64)
		if err != nil {
			return 0
		}
		return parsed
	default:
		return 0
	}
}

func hasFactID(records []FactRecord, id uint64) bool {
	for _, r := range records {
		if r.ID == id {
			return true
		}
	}
	return false
}

func pickDecisionText(decision FactDecision, fallback string) string {
	text := strings.TrimSpace(decision.Text)
	if text != "" {
		return text
	}
	return fallback
}

func splitTags(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		out = append(out, part)
	}
	return out
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

func mergeEventFacts(eventFacts []string, storedFacts []string) []string {
	if len(eventFacts) == 0 {
		return storedFacts
	}
	merged := make([]string, 0, len(eventFacts)+len(storedFacts))
	merged = append(merged, eventFacts...)
	merged = append(merged, storedFacts...)
	return dedupeStrings(merged)
}
