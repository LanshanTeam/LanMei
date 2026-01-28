package model

type GeminiModelConfig struct {
	BaseURL     string
	APIKey      string
	Model       string
	Temperature float32
	TopP        *float32
	TopK        *int
	MaxTokens   *int
	RetryTimes  int
}
