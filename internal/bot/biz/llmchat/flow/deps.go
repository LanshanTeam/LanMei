package flow

import (
	"LanMei/internal/bot/biz/llmchat/analysis"
	"LanMei/internal/bot/biz/llmchat/hooks"
	"LanMei/internal/bot/biz/llmchat/memory"
	"LanMei/internal/bot/utils/rerank"
	"LanMei/internal/bot/utils/websearch"

	fmodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/prompt"
)

type FrequencyController interface {
	ShouldThrottle(groupID string) bool
	MarkSent(groupID string)
}

type HookInfos struct {
	Chat   hooks.CallInfo
	Judge  hooks.CallInfo
	Plan   hooks.CallInfo
	Search hooks.CallInfo
}

type Dependencies struct {
	ChatModel      fmodel.BaseChatModel
	JudgeModel     fmodel.ToolCallingChatModel
	PlannerModel   fmodel.ToolCallingChatModel
	SearchModel    fmodel.BaseChatModel
	Template       *prompt.DefaultChatTemplate
	JudgeTemplate  *prompt.DefaultChatTemplate
	PlanTemplate   *prompt.DefaultChatTemplate
	SearchTemplate *prompt.DefaultChatTemplate
	InputAnalyzer  *analysis.InputAnalyzer
	Memory         *memory.MemoryManager
	Reranker       *rerank.Reranker
	Searcher       *websearch.Client
	Frequency      FrequencyController
	Hooks          *hooks.Runner
	HookInfos      HookInfos
}
