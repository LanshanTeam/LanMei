package model

import (
	arkmodel "github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"
)

type NodeConfig struct {
	Provider            Provider
	BaseURL             string
	Region              string
	APIKey              string
	Organization        string
	Project             string
	Model               string
	Temperature         float32
	TopP                *float32
	TopK                *int
	MaxTokens           *int
	MaxCompletionTokens *int
	PresencePenalty     *float32
	FrequencyPenalty    *float32
	Stop                []string
	APIVersion          string
	ByAzure             bool
	RetryTimes          int
	Thinking            *arkmodel.Thinking
}

func (c NodeConfig) ArkConfig() ArkModelConfig {
	return ArkModelConfig{
		BaseURL:         c.BaseURL,
		Region:          c.Region,
		APIKey:          c.APIKey,
		Model:           c.Model,
		Temperature:     c.Temperature,
		TopP:            c.TopP,
		MaxTokens:       c.MaxTokens,
		PresencePenalty: c.PresencePenalty,
		RetryTimes:      c.RetryTimes,
		Thinking:        c.Thinking,
	}
}

func (c NodeConfig) GeminiConfig() GeminiModelConfig {
	return GeminiModelConfig{
		BaseURL:     c.BaseURL,
		APIKey:      c.APIKey,
		Model:       c.Model,
		Temperature: c.Temperature,
		TopP:        c.TopP,
		TopK:        c.TopK,
		MaxTokens:   c.MaxTokens,
		RetryTimes:  c.RetryTimes,
	}
}

func (c NodeConfig) OpenAIConfig() OpenAIModelConfig {
	return OpenAIModelConfig{
		BaseURL:             c.BaseURL,
		APIKey:              c.APIKey,
		Organization:        c.Organization,
		Project:             c.Project,
		Model:               c.Model,
		Temperature:         c.Temperature,
		TopP:                c.TopP,
		MaxTokens:           c.MaxTokens,
		MaxCompletionTokens: c.MaxCompletionTokens,
		PresencePenalty:     c.PresencePenalty,
		FrequencyPenalty:    c.FrequencyPenalty,
		Stop:                c.Stop,
		APIVersion:          c.APIVersion,
		ByAzure:             c.ByAzure,
		RetryTimes:          c.RetryTimes,
	}
}

func LoadNodeConfig(node string) NodeConfig {
	cfg := NodeConfig{}
	cfg.Provider = DetectProvider(node)
	return loadLLMNodeConfig(node, cfg)
}

func loadLLMNodeConfig(node string, cfg NodeConfig) NodeConfig {
	prefix := nodeKey("LLM", node, "")
	if v := readString(prefix + "BaseURL"); v != "" {
		cfg.BaseURL = v
	}
	if v := readString(prefix + "Region"); v != "" {
		cfg.Region = v
	}
	if v := readString(prefix + "APIKey"); v != "" {
		cfg.APIKey = v
	}
	if v := readString(prefix + "Organization"); v != "" {
		cfg.Organization = v
	}
	if v := readString(prefix + "Project"); v != "" {
		cfg.Project = v
	}
	if v := readString(prefix + "Model"); v != "" {
		cfg.Model = v
	}
	if v, ok := readFloat32(prefix + "Temperature"); ok {
		cfg.Temperature = v
	}
	if v, ok := readFloat32(prefix + "TopP"); ok {
		cfg.TopP = &v
	}
	if v, ok := readInt(prefix + "TopK"); ok {
		cfg.TopK = &v
	}
	if v, ok := readInt(prefix + "MaxTokens"); ok {
		cfg.MaxTokens = &v
	}
	if v, ok := readInt(prefix + "MaxCompletionTokens"); ok {
		cfg.MaxCompletionTokens = &v
	}
	if v, ok := readFloat32(prefix + "PresencePenalty"); ok {
		cfg.PresencePenalty = &v
	}
	if v, ok := readFloat32(prefix + "FrequencyPenalty"); ok {
		cfg.FrequencyPenalty = &v
	}
	if v := readString(prefix + "APIVersion"); v != "" {
		cfg.APIVersion = v
	}
	if v, ok := readBool(prefix + "ByAzure"); ok {
		cfg.ByAzure = v
	}
	if v, ok := readStrings(prefix + "Stop"); ok {
		cfg.Stop = v
	}
	if v, ok := readInt(prefix + "RetryTimes"); ok {
		cfg.RetryTimes = v
	}
	if thinking := readThinking(prefix + "Thinking"); thinking != nil {
		cfg.Thinking = thinking
	}
	return cfg
}
