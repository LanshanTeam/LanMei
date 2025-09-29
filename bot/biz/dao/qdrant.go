package dao

import (
	"LanMei/bot/config"
	"LanMei/bot/utils/feishu"
	"LanMei/bot/utils/llog"
	"context"
	"errors"
	"math"
	"time"

	embed "github.com/cloudwego/eino-ext/components/embedding/ark"
	"github.com/qdrant/go-client/qdrant"
)

type QdrantManagerImpl struct {
	client   *qdrant.Client
	embedder *embed.Embedder
}

var CollectionName = "LanMei-Embed"

func NewQdrantManager() *QdrantManagerImpl {
	cli, err := qdrant.NewClient(&qdrant.Config{
		Host:   config.K.String("Database.Qdrant.Host"),
		Port:   config.K.Int("Database.Qdrant.Port"),
		APIKey: config.K.String("Database.Qdrant.APIKey"),
	})
	if err != nil {
		llog.Fatal("连接 qdrant 失败", err)
		return nil
	}

	ctx := context.Background()

	// 先查集合是否存在
	exists, err := cli.CollectionExists(ctx, CollectionName)
	if err != nil {
		llog.Fatal("连接 qdrant 失败", err)
		return nil
	}
	if !exists {
		createErr := cli.CreateCollection(ctx, &qdrant.CreateCollection{
			CollectionName: CollectionName,
			VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
				Size:     4096,
				Distance: qdrant.Distance_Cosine,
			}),
		})
		if createErr != nil {
			llog.Fatal("创建 qdrant 集合失败", createErr)
			return nil
		}
	}
	var RetryTimes int = 1
	embedder, err := embed.NewEmbedder(context.Background(), &embed.EmbeddingConfig{
		BaseURL:    config.K.String("Ark.BaseURL"),
		Region:     config.K.String("Ark.Region"),
		APIKey:     config.K.String("Ark.APIKey"),
		Model:      config.K.String("Ark.EmbedModel"),
		RetryTimes: &RetryTimes,
	})
	if err != nil {
		llog.Fatal("初始化向量模型失败", err)
		return nil
	}
	m := &QdrantManagerImpl{
		client:   cli,
		embedder: embedder,
	}
	return m
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

func (m *QdrantManagerImpl) UpdateKnowledge(ctx context.Context, kl *feishu.ReplyTable, colletion string) {
	datas := kl.GetKnowledge()
	if len(datas) == 0 {
		return
	}

	embeddings, err := m.embedder.EmbedStrings(ctx, datas)
	if err != nil {
		llog.Error("知识库向量化失败", err)
		return
	}
	if len(embeddings) != len(datas) {
		llog.Error("embedding 数量与文本数不一致", "embN", len(embeddings), "dataN", len(datas))
		return
	}

	points := make([]*qdrant.PointStruct, 0, len(datas))
	for i, vecF64 := range embeddings {
		vecF32, err := f64ToF32(vecF64)
		if err != nil {
			llog.Error("向量转换失败", "i", i, "err", err)
			continue
		}

		id := uint64(i + 1)

		payload := map[string]*qdrant.Value{
			"text": qdrant.NewValueList(&qdrant.ListValue{
				Values: []*qdrant.Value{qdrant.NewValueString(datas[i])},
			}),
		}

		points = append(points, &qdrant.PointStruct{
			Id:      qdrant.NewIDNum(id),
			Vectors: qdrant.NewVectors(vecF32...),
			Payload: payload,
		})
	}

	if len(points) == 0 {
		return
	}
	err = m.TruncateByScroll(ctx, colletion, 500)
	if err != nil {
		llog.Error("删除向量库数据失败", err)
		return
	}
	_, err = m.client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: colletion,
		Points:         points,
	})
	if err != nil {
		llog.Error("Qdrant Upsert 失败", err)
		return
	}
}

func valueMapToStrSlicePayload(m map[string]*qdrant.Value) map[string][]string {
	if m == nil {
		return nil
	}
	out := make(map[string][]string, len(m))
	for k, v := range m {
		lv, ok := v.GetKind().(*qdrant.Value_ListValue)
		if !ok || lv.ListValue == nil {
			continue
		}
		values := lv.ListValue.GetValues()
		ss := make([]string, 0, len(values))
		for _, item := range values {
			if sv, ok := item.GetKind().(*qdrant.Value_StringValue); ok {
				ss = append(ss, sv.StringValue)
			}
		}
		out[k] = ss
	}
	return out
}

func pointIDToAny(pid *qdrant.PointId) any {
	switch id := pid.GetPointIdOptions().(type) {
	case *qdrant.PointId_Num:
		return id.Num
	case *qdrant.PointId_Uuid:
		return id.Uuid
	default:
		return nil
	}
}

// ====== TopK：查询向量只收 float64，返回 payload 为 map[string][]string ======
func (m *QdrantManagerImpl) SearchTopKF64(ctx context.Context, collection string, query []float64, topK uint64) ([]SearchResult, error) {
	if topK == 0 {
		topK = 5
	}
	q32, err := f64ToF32(query)
	if err != nil {
		return nil, err
	}
	spList, err := m.client.Query(ctx, &qdrant.QueryPoints{
		CollectionName: collection,
		Query:          qdrant.NewQuery(q32...),
		Limit:          &topK,
		WithPayload:    qdrant.NewWithPayload(true),
	})
	if err != nil {
		return nil, err
	}

	out := make([]SearchResult, 0, len(spList))
	for _, sp := range spList {
		out = append(out, SearchResult{
			ID:      pointIDToAny(sp.GetId()),
			Score:   sp.GetScore(),
			Payload: valueMapToStrSlicePayload(sp.GetPayload()),
		})
	}
	return out, nil
}

func (m *QdrantManagerImpl) SearchTopKByText(ctx context.Context, collection string, text string, topK uint64) ([]SearchResult, error) {
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

func (m *QdrantManagerImpl) TruncateByScroll(ctx context.Context, name string, batch int) error {
	if batch <= 0 {
		batch = 1000
	}
	var offset *qdrant.PointId = nil

	for {
		records, err := m.client.Scroll(ctx, &qdrant.ScrollPoints{
			CollectionName: name,
			WithPayload:    qdrant.NewWithPayload(false),
			WithVectors:    qdrant.NewWithVectors(false),
			Limit:          ptr(uint32(batch)),
			Offset:         offset,
		})
		if err != nil {
			return err
		}
		if len(records) == 0 {
			return nil
		}

		ids := make([]*qdrant.PointId, 0, len(records))
		for _, r := range records {
			ids = append(ids, r.GetId())
		}

		if _, err := m.client.Delete(ctx, &qdrant.DeletePoints{
			CollectionName: name,
			Points: &qdrant.PointsSelector{
				PointsSelectorOneOf: &qdrant.PointsSelector_Points{
					Points: &qdrant.PointsIdsList{Ids: ids},
				},
			},
		}); err != nil {
			return err
		}

		offset = records[len(records)-1].GetId()
	}
}

func ptr[T any](v T) *T { return &v }

func (m *DBManagerImpl) DeleteEmbedding(ctx context.Context, collection string, batch int) {
	m.embedDB.TruncateByScroll(ctx, collection, 500)
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

func (m *DBManagerImpl) UpdateEmbedding(ctx context.Context, collection string, feishu *feishu.ReplyTable) {
	for {
		m.embedDB.UpdateKnowledge(ctx, feishu, collection)
		time.Sleep(30 * time.Second)
	}
}
