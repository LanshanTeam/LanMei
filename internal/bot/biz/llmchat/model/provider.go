package model

import "strings"

type Provider string

const (
	ProviderArk    Provider = "ark"
	ProviderGemini Provider = "gemini"
	ProviderOpenAI Provider = "openai"
)

func (p Provider) String() string {
	return string(p)
}

func NormalizeProvider(value string) Provider {
	v := strings.TrimSpace(strings.ToLower(value))
	switch v {
	case "ark", "volc", "volcengine", "arkruntime":
		return ProviderArk
	case "gemini", "google", "gcp":
		return ProviderGemini
	case "openai", "open-ai", "oa":
		return ProviderOpenAI
	default:
		return ""
	}
}

func DetectProvider(node string) Provider {
	if v := readString(
		nodeKey("LLM", node, "Type"),
		nodeKey("LLM", node, "type"),
		nodeKey("LLM", node, "Provider"),
		nodeKey("LLM", node, "provider"),
	); v != "" {
		if p := NormalizeProvider(v); p != "" {
			return p
		}
	}
	return ""
}
