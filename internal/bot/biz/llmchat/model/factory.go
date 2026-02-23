package model

import (
	"fmt"

	fmodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

func NewChatModel(cfg NodeConfig) (fmodel.BaseChatModel, error) {
	switch cfg.Provider {
	case ProviderGemini:
		return NewGeminiChatModel(cfg.GeminiConfig())
	case ProviderArk:
		return NewArkChatModel(cfg.ArkConfig())
	case ProviderOpenAI:
		return NewOpenAIChatModel(cfg.OpenAIConfig())
	default:
		return nil, fmt.Errorf("unknown model provider: %s", cfg.Provider)
	}
}

func NewToolCallingChatModel(cfg NodeConfig, tool *schema.ToolInfo) (fmodel.ToolCallingChatModel, error) {
	return NewToolCallingChatModelWithTools(cfg, toolsOrNil(tool))
}

func NewToolCallingChatModelWithTools(cfg NodeConfig, tools []*schema.ToolInfo) (fmodel.ToolCallingChatModel, error) {
	switch cfg.Provider {
	case ProviderGemini:
		base, err := NewGeminiChatModel(cfg.GeminiConfig())
		if err != nil {
			return nil, err
		}
		if len(tools) == 0 {
			return base, nil
		}
		return base.WithTools(tools)
	case ProviderArk:
		base, err := NewArkChatModel(cfg.ArkConfig())
		if err != nil {
			return nil, err
		}
		if len(tools) == 0 {
			return base, nil
		}
		return base.WithTools(tools)
	case ProviderOpenAI:
		base, err := NewOpenAIChatModel(cfg.OpenAIConfig())
		if err != nil {
			return nil, err
		}
		if len(tools) == 0 {
			return base, nil
		}
		return base.WithTools(tools)
	default:
		return nil, fmt.Errorf("unknown model provider: %s", cfg.Provider)
	}
}

func toolsOrNil(tool *schema.ToolInfo) []*schema.ToolInfo {
	if tool == nil {
		return nil
	}
	return []*schema.ToolInfo{tool}
}
