package model

import (
	"context"

	"github.com/cloudwego/eino-ext/components/model/gemini"
	fmodel "github.com/cloudwego/eino/components/model"
	"google.golang.org/genai"
)

func NewGeminiChatModel(cfg GeminiModelConfig) (fmodel.ToolCallingChatModel, error) {
	clientCfg := &genai.ClientConfig{
		APIKey:  cfg.APIKey,
		Backend: genai.BackendGeminiAPI,
	}
	if cfg.BaseURL != "" {
		clientCfg.HTTPOptions.BaseURL = cfg.BaseURL
	}
	client, err := genai.NewClient(context.Background(), clientCfg)
	if err != nil {
		return nil, err
	}
	modelCfg := &gemini.Config{
		Client:      client,
		Model:       cfg.Model,
		MaxTokens:   cfg.MaxTokens,
		Temperature: floatPtr(cfg.Temperature),
		TopP:        cfg.TopP,
		TopK:        toInt32Ptr(cfg.TopK),
	}
	return gemini.NewChatModel(context.Background(), modelCfg)
}

func toInt32Ptr(value *int) *int32 {
	if value == nil {
		return nil
	}
	converted := int32(*value)
	return &converted
}
