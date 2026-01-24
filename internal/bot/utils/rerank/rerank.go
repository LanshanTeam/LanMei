package rerank

import (
	"LanMei/internal/bot/utils/llog"
	"bytes"
	"encoding/json"
	"net/http"
	"sort"
	"time"
)

// 由于 eino 没有 rerank，自己包个
type Reranker struct {
	APIKey  string
	Model   string
	BaseUrl string
}

func NewReranker(APIKey, Model, BaseUrl string) *Reranker {
	return &Reranker{
		APIKey:  APIKey,
		Model:   Model,
		BaseUrl: BaseUrl,
	}
}

type rerankRequest struct {
	Model           string   `json:"model"`
	Query           string   `json:"query"`
	Documents       []string `json:"documents"`
	ReturnDocuments bool     `json:"return_documents"`
}

type rerankResponse struct {
	Model   string         `json:"model"`
	Results []rerankResult `json:"results"`
	Usage   *rerankUsage   `json:"usage,omitempty"`
}

type rerankResult struct {
	Index          int             `json:"index"`
	RelevanceScore float64         `json:"relavance_score"`
	Document       *rerankDocument `json:"document,omitempty"`
}

type rerankDocument struct {
	Text string `json:"text"`
}

type rerankUsage struct {
	PromptTokens int `json:"prompt_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// TopN 基于大模型重排，返回分数最高的前 N 个文档文本（降序）。
// 若请求失败或响应异常，返回空切片。
func (r *Reranker) TopN(N int, docs []string, input string) []string {
	if len(docs) == 0 || input == "" || r.APIKey == "" || r.Model == "" || r.BaseUrl == "" {
		return []string{}
	}
	if N <= 0 {
		N = 1
	}
	if N > len(docs) {
		N = len(docs)
	}

	reqBody := rerankRequest{
		Model:           r.Model,
		Query:           input,
		Documents:       docs,
		ReturnDocuments: true,
	}
	raw, err := json.Marshal(reqBody)
	if err != nil {
		return []string{}
	}

	httpClient := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest(http.MethodPost, r.BaseUrl, bytes.NewReader(raw))
	if err != nil {
		return []string{}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+r.APIKey)

	resp, err := httpClient.Do(req)
	if err != nil {
		return []string{}
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		llog.Error("请求失败，状态码：", resp.StatusCode)
		return []string{}
	}

	var rr rerankResponse
	if err := json.NewDecoder(resp.Body).Decode(&rr); err != nil {
		return []string{}
	}
	if len(rr.Results) == 0 {
		return []string{}
	}

	sort.Slice(rr.Results, func(i, j int) bool {
		return rr.Results[i].RelevanceScore > rr.Results[j].RelevanceScore
	})

	// 取 Top-N
	out := make([]string, 0, N)
	for i := 0; i < N && i < len(rr.Results); i++ {
		res := rr.Results[i]
		if res.Document != nil && res.Document.Text != "" {
			out = append(out, res.Document.Text)
		}
	}
	return out
}
