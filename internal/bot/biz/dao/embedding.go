package dao

import (
	"LanMei/internal/bot/config"
	"LanMei/internal/bot/utils/feishu"
	"LanMei/internal/bot/utils/llog"
	"context"
	"encoding/json"
	"errors"
	"math"
	"strings"

	embed "github.com/cloudwego/eino-ext/components/embedding/ark"
	"github.com/pgvector/pgvector-go"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type EmbeddingManagerImpl struct {
	db       *gorm.DB
	embedder *embed.Embedder
}

type EmbeddingRecord struct {
	Collection  string          `gorm:"column:collection;primaryKey;type:text"`
	ID          uint64          `gorm:"column:id;primaryKey"`
	Embedding   pgvector.Vector `gorm:"column:embedding;type:vector(4096)"`
	PayloadJSON json.RawMessage `gorm:"column:payload_json;type:jsonb"`
}

func (EmbeddingRecord) TableName() string {
	return "embeddings"
}

var CollectionName = "LanMei-Embed"

func NewEmbeddingManager(db *gorm.DB) *EmbeddingManagerImpl {
	cfg := loadEmbedConfig()
	embedder, err := embed.NewEmbedder(context.Background(), cfg)
	if err != nil {
		llog.Fatal("初始化向量模型失败", err)
		return nil
	}
	m := &EmbeddingManagerImpl{
		db:       db,
		embedder: embedder,
	}
	return m
}

func loadEmbedConfig() *embed.EmbeddingConfig {
	retryTimes := 1
	if config.K != nil {
		if config.K.Exists("Ark.Embed.RetryTimes") {
			retryTimes = config.K.Int("Ark.Embed.RetryTimes")
		} else if config.K.Exists("Ark.RetryTimes") {
			retryTimes = config.K.Int("Ark.RetryTimes")
		}
	}
	baseURL := readConfigString("Ark.Embed.BaseURL", "Ark.BaseURL")
	region := readConfigString("Ark.Embed.Region", "Ark.Region")
	apiKey := readConfigString("Ark.Embed.APIKey", "Ark.APIKey")
	model := readConfigString("Ark.Embed.Model", "Ark.EmbedModel", "Ark.Model")
	return &embed.EmbeddingConfig{
		BaseURL:    baseURL,
		Region:     region,
		APIKey:     apiKey,
		Model:      model,
		RetryTimes: &retryTimes,
	}
}

func readConfigString(keys ...string) string {
	if config.K == nil {
		return ""
	}
	for _, key := range keys {
		if key == "" {
			continue
		}
		value := strings.TrimSpace(config.K.String(key))
		if value != "" {
			return value
		}
	}
	return ""
}

type PointF64 struct {
	ID      uint64
	Vector  []float64
	Payload map[string][]string
}

type SearchResult struct {
	ID      uint64
	Score   float32
	Payload map[string][]string
	Text    string
}

type EmbeddingItem struct {
	ID   uint64
	Text string
}

func f64ToF32(vec []float64) ([]float32, error) {
	if len(vec) == 0 {
		return nil, errors.New("vector is empty")
	}
	out := make([]float32, len(vec))
	for i, v := range vec {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return nil, errors.New("vector contains NaN or Inf")
		}
		out[i] = float32(v)
	}
	return out, nil
}

func (m *EmbeddingManagerImpl) UpdateKnowledge(ctx context.Context, datas []feishu.KeyValue, collection string) {
	// llog.Debug("", datas)
	if len(datas) == 0 {
		return
	}
	items := make([]EmbeddingItem, 0, len(datas))
	for _, data := range datas {
		items = append(items, EmbeddingItem{
			ID:   uint64(data.Key),
			Text: data.Value,
		})
	}
	if err := m.UpsertTextItems(ctx, collection, items); err != nil {
		llog.Error("更新向量库失败", err)
	}
}

func (m *EmbeddingManagerImpl) UpsertTextItems(ctx context.Context, collection string, items []EmbeddingItem) error {
	if len(items) == 0 {
		return nil
	}
	filtered := make([]EmbeddingItem, 0, len(items))
	strs := make([]string, 0, len(items))
	for _, item := range items {
		if strings.TrimSpace(item.Text) == "" {
			continue
		}
		filtered = append(filtered, item)
		strs = append(strs, item.Text)
	}
	if len(filtered) == 0 {
		return nil
	}
	embeddings, err := m.embedder.EmbedStrings(ctx, strs)
	if err != nil {
		return err
	}
	if len(embeddings) != len(filtered) {
		return errors.New("embedding count mismatch")
	}
	records := make([]EmbeddingRecord, 0, len(filtered))
	for i, item := range filtered {
		vecF64 := embeddings[i]
		vecF32, err := f64ToF32(vecF64)
		if err != nil {
			llog.Error("向量转换失败", "err", err)
			continue
		}
		payload, err := json.Marshal([]string{item.Text})
		if err != nil {
			llog.Error("payload 序列化失败", "err", err)
			continue
		}
		records = append(records, EmbeddingRecord{
			Collection:  collection,
			ID:          item.ID,
			Embedding:   pgvector.NewVector(vecF32),
			PayloadJSON: payload,
		})
	}
	if len(records) == 0 {
		return nil
	}
	db := m.db.WithContext(ctx)
	return db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "collection"}, {Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"embedding", "payload_json"}),
	}).CreateInBatches(records, 200).Error
}

// ====== TopK：查询向量只收 float64，返回 payload 为 map[string][]string ======
func (m *EmbeddingManagerImpl) SearchTopKF64(ctx context.Context, collection string, query []float64, topK uint64) ([]SearchResult, error) {
	if topK == 0 {
		topK = 5
	}
	q32, err := f64ToF32(query)
	if err != nil {
		return nil, err
	}
	vec := pgvector.NewVector(q32)
	type row struct {
		ID          uint64          `gorm:"column:id"`
		PayloadJSON json.RawMessage `gorm:"column:payload_json"`
		Distance    float64         `gorm:"column:distance"`
	}
	var rows []row
	err = m.db.WithContext(ctx).Raw(`SELECT id, payload_json, embedding <=> ? AS distance
		FROM embeddings WHERE collection = ? ORDER BY distance LIMIT ?`, vec, collection, topK).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]SearchResult, 0, len(rows))
	for _, r := range rows {
		var payload []string
		if len(r.PayloadJSON) > 0 {
			if err := json.Unmarshal(r.PayloadJSON, &payload); err != nil {
				llog.Error("payload 反序列化失败", "err", err)
			}
		}
		text := ""
		if len(payload) > 0 {
			text = payload[0]
		}
		score := float32(1.0 - r.Distance)
		out = append(out, SearchResult{
			ID:      r.ID,
			Score:   score,
			Payload: map[string][]string{"text": payload},
			Text:    text,
		})
	}
	return out, nil
}

func (m *EmbeddingManagerImpl) SearchTopKByText(ctx context.Context, collection string, text string, topK uint64) ([]SearchResult, error) {
	if text == "" {
		return nil, errors.New("query text is empty")
	}
	embs, err := m.embedder.EmbedStrings(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(embs) == 0 {
		return nil, errors.New("empty embedding result")
	}
	return m.SearchTopKF64(ctx, collection, embs[0], topK)
}

func (m *DBManagerImpl) DeleteEmbedding(ctx context.Context, collection string, batch int) {
	db := m.embedDB.db.WithContext(ctx)
	if batch <= 0 {
		if err := db.Exec(`DELETE FROM embeddings WHERE collection = ?`, collection).Error; err != nil {
			llog.Error("清空向量库失败", err)
		}
		return
	}
	for {
		res := db.Exec(`DELETE FROM embeddings WHERE ctid IN (
			SELECT ctid FROM embeddings WHERE collection = ? LIMIT ?
		)`, collection, batch)
		if res.Error != nil {
			llog.Error("清空向量库失败", res.Error)
			return
		}
		if res.RowsAffected == 0 {
			break
		}
	}
}

func (m *DBManagerImpl) GetTopK(ctx context.Context, collection string, K uint64, text string) []string {
	res, err := m.embedDB.SearchTopKByText(ctx, collection, text, K)
	if err != nil {
		llog.Error("查找向量库失败", err)
		return nil
	}
	out := make([]string, 0, len(res))
	for _, item := range res {
		if item.Payload == nil {
			continue
		}
		if texts, ok := item.Payload["text"]; ok && len(texts) > 0 {
			out = append(out, texts...)
		}
	}
	return out
}

func (m *DBManagerImpl) SearchTopKResults(ctx context.Context, collection string, text string, K uint64) []SearchResult {
	if m == nil || m.embedDB == nil {
		return nil
	}
	res, err := m.embedDB.SearchTopKByText(ctx, collection, text, K)
	if err != nil {
		llog.Error("查找向量库失败", err)
		return nil
	}
	return res
}

func (m *DBManagerImpl) DeleteEmbeddingByID(ctx context.Context, collection string, id uint64) error {
	if m == nil || m.embedDB == nil {
		return nil
	}
	if id == 0 {
		return nil
	}
	db := m.embedDB.db.WithContext(ctx)
	return db.Exec(`DELETE FROM embeddings WHERE collection = ? AND id = ?`, collection, id).Error
}

func (m *DBManagerImpl) UpsertEmbeddingTexts(ctx context.Context, collection string, items []EmbeddingItem) error {
	return m.embedDB.UpsertTextItems(ctx, collection, items)
}

func (m *DBManagerImpl) UpdateEmbedding(ctx context.Context, collection string, feishu *feishu.ReplyTable) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		datas := feishu.Wait()
		if ctx.Err() != nil {
			return
		}
		m.embedDB.UpdateKnowledge(ctx, datas, collection)
	}
}
