package embedding

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"LanMei/internal/bot/utils/llog"
)

type Provider string

const (
	ProviderZhipu Provider = "zhipu"
	ProviderArk   Provider = "ark"
)

type Config struct {
	Provider   Provider
	APIKey     string
	BaseURL    string
	Model      string
	Dimensions int
	Timeout    time.Duration
}

type Embedder interface {
	EmbedStrings(ctx context.Context, texts []string) ([][]float64, error)
}

func NewEmbedder(cfg *Config) (Embedder, error) {
	if cfg == nil {
		return nil, errors.New("embedding config is nil")
	}
	if cfg.APIKey == "" {
		return nil, errors.New("embedding APIKey is empty")
	}
	if cfg.Model == "" {
		return nil, errors.New("embedding Model is empty")
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}

	switch cfg.Provider {
	case ProviderZhipu:
		return newZhipuEmbedder(cfg)
	default:
		return nil, fmt.Errorf("unsupported embedding provider: %s", cfg.Provider)
	}
}

type zhipuEmbedder struct {
	apiKey     string
	baseURL    string
	model      string
	dimensions int
	timeout    time.Duration
	httpClient *http.Client
}

func newZhipuEmbedder(cfg *Config) (*zhipuEmbedder, error) {
	baseURL := strings.TrimSuffix(cfg.BaseURL, "/")
	if baseURL == "" {
		baseURL = "https://open.bigmodel.cn/api/paas/v4/embeddings"
	}
	return &zhipuEmbedder{
		apiKey:     cfg.APIKey,
		baseURL:    baseURL,
		model:      cfg.Model,
		dimensions: cfg.Dimensions,
		timeout:    cfg.Timeout,
		httpClient: &http.Client{Timeout: cfg.Timeout},
	}, nil
}

type zhipuRequest struct {
	Model      string   `json:"model"`
	Input      []string `json:"input"`
	Dimensions int      `json:"dimensions,omitempty"`
}

type zhipuResponse struct {
	Model string `json:"model"`
	Data  []struct {
		Index     int       `json:"index"`
		Object    string    `json:"object"`
		Embedding []float64 `json:"embedding"`
	} `json:"data"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func (e *zhipuEmbedder) EmbedStrings(ctx context.Context, texts []string) ([][]float64, error) {
	if len(texts) == 0 {
		return nil, errors.New("empty texts")
	}

	reqBody := zhipuRequest{
		Model: e.model,
		Input: texts,
	}
	if e.dimensions > 0 {
		reqBody.Dimensions = e.dimensions
	}

	raw, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request failed: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.baseURL, bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e.apiKey)

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response failed: %w", err)
	}

	var zr zhipuResponse
	if err := json.Unmarshal(body, &zr); err != nil {
		return nil, fmt.Errorf("unmarshal response failed: %w", err)
	}

	if zr.Error != nil {
		return nil, fmt.Errorf("API error: %s - %s", zr.Error.Code, zr.Error.Message)
	}

	if len(zr.Data) == 0 {
		return nil, errors.New("empty embedding result")
	}

	embeddings := make([][]float64, len(texts))
	for i := range embeddings {
		embeddings[i] = nil
	}
	for _, item := range zr.Data {
		if item.Index >= 0 && item.Index < len(embeddings) {
			embeddings[item.Index] = item.Embedding
		}
	}

	for i, emb := range embeddings {
		if emb == nil {
			llog.Debug(fmt.Sprintf("embedding missing for index %d", i))
		}
	}

	llog.Debug(fmt.Sprintf("embedding usage: prompt=%d, total=%d", zr.Usage.PromptTokens, zr.Usage.TotalTokens))
	return embeddings, nil
}
