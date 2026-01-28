package model

type OpenAIModelConfig struct {
	BaseURL             string
	APIKey              string
	Organization        string
	Project             string
	Model               string
	Temperature         float32
	TopP                *float32
	MaxTokens           *int
	MaxCompletionTokens *int
	PresencePenalty     *float32
	FrequencyPenalty    *float32
	Stop                []string
	APIVersion          string
	ByAzure             bool
	RetryTimes          int
}
