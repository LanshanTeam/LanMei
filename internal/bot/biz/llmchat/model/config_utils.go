package model

import (
	"LanMei/internal/bot/config"
	"strings"

	arkmodel "github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"
)

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

func readBool(keys ...string) (bool, bool) {
	for _, key := range keys {
		if key == "" || config.K == nil {
			continue
		}
		if !config.K.Exists(key) {
			continue
		}
		return config.K.Bool(key), true
	}
	return false, false
}

func readStrings(keys ...string) ([]string, bool) {
	for _, key := range keys {
		if key == "" || config.K == nil {
			continue
		}
		if !config.K.Exists(key) {
			continue
		}
		values := config.K.Strings(key)
		if len(values) == 0 {
			return nil, false
		}
		return values, true
	}
	return nil, false
}

func readThinking(keys ...string) *arkmodel.Thinking {
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
			return &arkmodel.Thinking{Type: arkmodel.ThinkingTypeEnabled}
		case "disabled", "disable", "false", "0", "no", "off":
			return &arkmodel.Thinking{Type: arkmodel.ThinkingTypeDisabled}
		default:
			continue
		}
	}
	return nil
}

func nodeKey(prefix, node, key string) string {
	if prefix == "" {
		return ""
	}
	if node == "" {
		if key == "" {
			return prefix + "."
		}
		return prefix + "." + key
	}
	if key == "" {
		return prefix + ".Nodes." + node + "."
	}
	return prefix + ".Nodes." + node + "." + key
}

func floatPtr(value float32) *float32 {
	return &value
}
