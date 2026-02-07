package types

import (
	"context"

	"LanMei/internal/bot/biz/llmchat/flow/hooks"
	"LanMei/internal/bot/biz/llmchat/memory"
	"LanMei/internal/bot/utils/rerank"
	"LanMei/internal/bot/utils/websearch"

	fmodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
)

type FrequencyController interface {
	ShouldThrottle(groupID string) bool
	MarkSent(groupID string)
}

type InputAnalyzer interface {
	Analyze(ctx context.Context, nickname, input string, history []schema.Message, knownFacts []string, userProfile string) (InputAnalysis, bool)
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
	InputAnalyzer  InputAnalyzer
	Memory         *memory.MemoryManager
	Reranker       *rerank.Reranker
	Searcher       *websearch.Client
	Frequency      FrequencyController
	Hooks          *hooks.Runner
	HookInfos      HookInfos
}
