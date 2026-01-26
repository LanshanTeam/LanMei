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
)

type EmbeddingManagerImpl struct {
	db       *gorm.DB
	embedder *embed.Embedder
}

type EmbeddingRecord struct {
	Collection  string          `gorm:"column:collection;primaryKey;type:text"`
	ID          uint64          `gorm:"column:id;primaryKey"`
	Embedding   pgvector.Vector `gorm:"column:embedding;type:vector(4096)"`
	PayloadJSON string          `gorm:"column:payload_json;type:jsonb"`
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
	ID      any
	Score   float32
	Payload map[string][]string
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

func (m *EmbeddingManagerImpl) UpdateKnowledge(ctx context.Context, datas []feishu.KeyValue, colletion string) {
	llog.Debug("", datas)
	if len(datas) == 0 {
		return
	}
	strs := make([]string, 0, len(datas))
	for _, data := range datas {
		strs = append(strs, data.Value)
	}
	embeddings, err := m.embedder.EmbedStrings(ctx, strs)
	if err != nil {
		llog.Error("知识库向量化失败", err)
		return
	}
	if len(embeddings) != len(datas) {
		llog.Error("embedding 数量与文本数不一致", "embN", len(embeddings), "dataN", len(datas))
		return
	}

	db := m.db.WithContext(ctx)
	for i, vecF64 := range embeddings {
		vecF32, err := f64ToF32(vecF64)
		if err != nil {
			llog.Error("向量转换失败", "i", i, "err", err)
			continue
		}

		payload, err := json.Marshal([]string{datas[i].Value})
		if err != nil {
			llog.Error("payload 序列化失败", "i", i, "err", err)
			continue
		}
		vec := pgvector.NewVector(vecF32)
		if err := db.Exec(`INSERT INTO embeddings (collection, id, embedding, payload_json)
			VALUES (?, ?, ?, ?)
			ON CONFLICT (collection, id) DO UPDATE SET embedding = EXCLUDED.embedding, payload_json = EXCLUDED.payload_json`,
			colletion, uint64(datas[i].Key), vec, string(payload)).Error; err != nil {
			llog.Error("Postgres Upsert 失败", err)
			return
		}
	}
}

func (m *EmbeddingManagerImpl) UpsertTextItems(ctx context.Context, collection string, items []EmbeddingItem) error {
	if len(items) == 0 {
		return nil
	}
	strs := make([]string, 0, len(items))
	for _, item := range items {
		if item.Text == "" {
			continue
		}
		strs = append(strs, item.Text)
	}
	if len(strs) == 0 {
		return nil
	}
	embeddings, err := m.embedder.EmbedStrings(ctx, strs)
	if err != nil {
		return err
	}
	if len(embeddings) != len(strs) {
		return errors.New("embedding count mismatch")
	}
	db := m.db.WithContext(ctx)
	idx := 0
	for _, item := range items {
		if item.Text == "" {
			continue
		}
		vecF64 := embeddings[idx]
		idx++
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
		vec := pgvector.NewVector(vecF32)
		if err := db.Exec(`INSERT INTO embeddings (collection, id, embedding, payload_json)
			VALUES (?, ?, ?, ?)
			ON CONFLICT (collection, id) DO UPDATE SET embedding = EXCLUDED.embedding, payload_json = EXCLUDED.payload_json`,
			collection, item.ID, vec, string(payload)).Error; err != nil {
			llog.Error("Postgres Upsert 失败", err)
			return err
		}
	}
	return nil
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
		ID          uint64  `gorm:"column:id"`
		PayloadJSON string  `gorm:"column:payload_json"`
		Distance    float64 `gorm:"column:distance"`
	}
	var rows []row
	err = m.db.WithContext(ctx).Raw(`SELECT id, payload_json, embedding <=> ? AS distance
		FROM embeddings WHERE collection = ? ORDER BY embedding <=> ? LIMIT ?`, vec, collection, vec, topK).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]SearchResult, 0, len(rows))
	for _, r := range rows {
		var payload []string
		if r.PayloadJSON != "" {
			if err := json.Unmarshal([]byte(r.PayloadJSON), &payload); err != nil {
				llog.Error("payload 反序列化失败", "err", err)
			}
		}
		score := float32(1.0 - r.Distance)
		out = append(out, SearchResult{
			ID:      r.ID,
			Score:   score,
			Payload: map[string][]string{"text": payload},
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
	if err := m.embedDB.db.WithContext(ctx).Exec(`DELETE FROM embeddings WHERE collection = ?`, collection).Error; err != nil {
		llog.Error("清空向量库失败", err)
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

func (m *DBManagerImpl) UpsertEmbeddingTexts(ctx context.Context, collection string, items []EmbeddingItem) error {
	return m.embedDB.UpsertTextItems(ctx, collection, items)
}

func (m *DBManagerImpl) UpdateEmbedding(ctx context.Context, collection string, feishu *feishu.ReplyTable) {
	for {
		datas := feishu.Wait()
		m.embedDB.UpdateKnowledge(ctx, datas, collection)
	}
}
