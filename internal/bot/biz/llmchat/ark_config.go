package llmchat

import (
	"LanMei/internal/bot/config"
	"strings"

	"github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"
)

type ArkModelConfig struct {
	BaseURL         string
	Region          string
	APIKey          string
	Model           string
	Temperature     float32
	PresencePenalty *float32
	RetryTimes      int
	Thinking        *model.Thinking
}

func loadArkNodeConfig(node string, defaults ArkModelConfig) ArkModelConfig {
	cfg := defaults
	if config.K == nil {
		return cfg
	}
	prefix := ""
	if node != "" {
		prefix = "Ark.Nodes." + node + "."
	}
	if v := readString(prefix+"BaseURL", "Ark.BaseURL"); v != "" {
		cfg.BaseURL = v
	}
	if v := readString(prefix+"Region", "Ark.Region"); v != "" {
		cfg.Region = v
	}
	if v := readString(prefix+"APIKey", "Ark.APIKey"); v != "" {
		cfg.APIKey = v
	}
	if v := readString(prefix+"Model", "Ark.Model"); v != "" {
		cfg.Model = v
	}
	if v, ok := readFloat32(prefix+"Temperature", "Ark.Temperature"); ok {
		cfg.Temperature = v
	}
	if v, ok := readFloat32(prefix+"PresencePenalty", "Ark.PresencePenalty"); ok {
		cfg.PresencePenalty = &v
	}
	if v, ok := readInt(prefix+"RetryTimes", "Ark.RetryTimes"); ok {
		cfg.RetryTimes = v
	}
	if thinking := readThinking(prefix+"Thinking", "Ark.Thinking"); thinking != nil {
		cfg.Thinking = thinking
	}
	return cfg
}

func readString(keys ...string) string {
	for _, key := range keys {
		if key == "" || config.K == nil {
			continue
		}
		v := strings.TrimSpace(config.K.String(key))
		if v != "" {
			return v
		}
	}
	return ""
}

func readFloat32(keys ...string) (float32, bool) {
	for _, key := range keys {
		if key == "" || config.K == nil {
			continue
		}
		if !config.K.Exists(key) {
			continue
		}
		return float32(config.K.Float64(key)), true
	}
	return 0, false
}

func readInt(keys ...string) (int, bool) {
	for _, key := range keys {
		if key == "" || config.K == nil {
			continue
		}
		if !config.K.Exists(key) {
			continue
		}
		return config.K.Int(key), true
	}
	return 0, false
}

func readThinking(keys ...string) *model.Thinking {
	for _, key := range keys {
		if key == "" || config.K == nil {
			continue
		}
		if !config.K.Exists(key) {
			continue
		}
		value := strings.TrimSpace(strings.ToLower(config.K.String(key)))
		if value == "" {
			continue
		}
		switch value {
		case "enabled", "enable", "true", "1", "yes", "on":
			return &model.Thinking{Type: model.ThinkingTypeEnabled}
		case "disabled", "disable", "false", "0", "no", "off":
			return &model.Thinking{Type: model.ThinkingTypeDisabled}
		default:
			continue
		}
	}
	return nil
}
