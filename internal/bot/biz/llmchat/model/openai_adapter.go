package model

import (
	"context"

	openaimodel "github.com/cloudwego/eino-ext/components/model/openai"
	fmodel "github.com/cloudwego/eino/components/model"
)

func NewOpenAIChatModel(cfg OpenAIModelConfig) (fmodel.ToolCallingChatModel, error) {
	modelCfg := &openaimodel.ChatModelConfig{
		ByAzure:             cfg.ByAzure,
		BaseURL:             cfg.BaseURL,
		APIVersion:          cfg.APIVersion,
		APIKey:              cfg.APIKey,
		Model:               cfg.Model,
		MaxTokens:           cfg.MaxTokens,
		MaxCompletionTokens: cfg.MaxCompletionTokens,
		Temperature:         floatPtr(cfg.Temperature),
		TopP:                cfg.TopP,
		Stop:                cfg.Stop,
		PresencePenalty:     cfg.PresencePenalty,
		FrequencyPenalty:    cfg.FrequencyPenalty,
	}
	return openaimodel.NewChatModel(context.Background(), modelCfg)
}
