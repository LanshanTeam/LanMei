package types

import (
	"LanMei/internal/bot/biz/llmchat/memory"
	"LanMei/internal/bot/biz/llmchat/reactlog"

	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/flow/agent/react"
)

type FrequencyController interface {
	ShouldThrottle(groupID string) bool
	MarkSent(groupID string)
}

type Dependencies struct {
	ReActAgent          *react.Agent
	ReActTemplate       *prompt.DefaultChatTemplate
	SkillPromptInjector func(string) string
	Memory              *memory.MemoryManager
	Frequency           FrequencyController
	ReActLog            *reactlog.Window
}
