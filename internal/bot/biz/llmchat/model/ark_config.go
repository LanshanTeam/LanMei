package model

import (
	"context"

	"github.com/cloudwego/eino-ext/components/model/ark"
	arkmodel "github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"
)

type ArkModelConfig struct {
	BaseURL         string
	Region          string
	APIKey          string
	Model           string
	Temperature     float32
	TopP            *float32
	MaxTokens       *int
	PresencePenalty *float32
	RetryTimes      int
	Thinking        *arkmodel.Thinking
}

func NewArkChatModel(modelCfg ArkModelConfig) (*ark.ChatModel, error) {
	retryTimes := modelCfg.RetryTimes
	if retryTimes <= 0 {
		retryTimes = 1
	}
	arkCfg := &ark.ChatModelConfig{
		BaseURL:     modelCfg.BaseURL,
		Region:      modelCfg.Region,
		APIKey:      modelCfg.APIKey,
		Model:       modelCfg.Model,
		Temperature: &modelCfg.Temperature,
		RetryTimes:  &retryTimes,
	}
	if modelCfg.PresencePenalty != nil {
		arkCfg.PresencePenalty = modelCfg.PresencePenalty
	}
	if modelCfg.TopP != nil {
		arkCfg.TopP = modelCfg.TopP
	}
	if modelCfg.MaxTokens != nil {
		arkCfg.MaxTokens = modelCfg.MaxTokens
	}
	if modelCfg.Thinking != nil {
		arkCfg.Thinking = modelCfg.Thinking
	}
	return ark.NewChatModel(context.Background(), arkCfg)
}
