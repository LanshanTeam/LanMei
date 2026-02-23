package llmchat

import (
	"context"
	"time"

	"LanMei/internal/bot/biz/dao"
	"LanMei/internal/bot/biz/llmchat/flow"
	"LanMei/internal/bot/biz/llmchat/flow/hooks"
	llmtemplate "LanMei/internal/bot/biz/llmchat/flow/template"
	flowtypes "LanMei/internal/bot/biz/llmchat/flow/types"
	"LanMei/internal/bot/biz/llmchat/memory"
	llmmodel "LanMei/internal/bot/biz/llmchat/model"
	reactbiz "LanMei/internal/bot/biz/llmchat/react"
	"LanMei/internal/bot/biz/llmchat/reactlog"
	"LanMei/internal/bot/config"
	"LanMei/internal/bot/utils/feishu"
	"LanMei/internal/bot/utils/llog"
	"LanMei/internal/bot/utils/rerank"
	"LanMei/internal/bot/utils/websearch"

	"github.com/cloudwego/eino/compose"
	reactagent "github.com/cloudwego/eino/flow/agent/react"
)

const (
	MaxHistory int = 20
)

type ChatEngine struct {
	ReplyTable *feishu.ReplyTable
	memory     *memory.MemoryManager
	frequency  *FrequencyControlManager
	hooks      *hooks.Runner
	flow       *flow.ChatFlow
}

func NewChatEngine() *ChatEngine {
	chatConfig := mustLoadNodeConfig("Chat")
	memoryConfig := mustLoadNodeConfig("Memory")

	reactModel, err := llmmodel.NewToolCallingChatModel(chatConfig, nil)
	if err != nil {
		llog.Fatal("初始化 ReAct 模型", err)
		return nil
	}
	factModel, err := llmmodel.NewToolCallingChatModel(memoryConfig, memory.BuildFactTool())
	if err != nil {
		llog.Fatal("初始化 memory 事实抽取工具失败", err)
		return nil
	}
	factUpdateModel, err := llmmodel.NewToolCallingChatModel(memoryConfig, memory.BuildFactUpdateTool())
	if err != nil {
		llog.Fatal("初始化 memory 事实更新工具失败", err)
		return nil
	}
	profileModel, err := llmmodel.NewToolCallingChatModel(memoryConfig, memory.BuildProfileTool())
	if err != nil {
		llog.Fatal("初始化 memory 画像工具失败", err)
		return nil
	}

	reactTemplate := llmtemplate.BuildReActTemplate()
	hookRunner := hooks.NewRunner(hooks.NewDurationLogger())
	factHookInfo := hooks.CallInfo{Node: "fact_extract", Model: memoryConfig.Model}
	factUpdateHookInfo := hooks.CallInfo{Node: "fact_update", Model: memoryConfig.Model}
	profileHookInfo := hooks.CallInfo{Node: "profile", Model: memoryConfig.Model}

	reranker := rerank.NewReranker(
		config.K.String("Infini.APIKey"),
		config.K.String("Infini.Model"),
		config.K.String("Infini.BaseURL"),
	)
	reply := feishu.NewReplyTable()
	go dao.DBManager.UpdateEmbedding(context.Background(), dao.CollectionName, reply)

	factExtractor := memory.NewFactExtractor(factModel, hookRunner, factHookInfo)
	factUpdater := memory.NewFactUpdater(factUpdateModel, hookRunner, factUpdateHookInfo)
	profileUpdater := memory.NewProfileUpdater(profileModel, hookRunner, profileHookInfo)
	memoryManager := memory.NewMemoryManager(reranker, factExtractor, factUpdater, profileUpdater, MaxHistory)
	memoryWorker := memory.NewMemoryWorker(memoryManager, 12*time.Second, 4, 12)
	memoryWorker.Start()
	memoryManager.BindWorker(memoryWorker)
	searcher := websearch.NewClient()
	frequencyManager := NewFrequencyControlManager()

	reactLog := reactlog.NewWindow(0, 0, 0)
	_, skillsMiddleware, skillTools := reactbiz.InitSkills()
	reactTools, err := reactbiz.BuildTools(searcher, memoryManager, reranker)
	if err != nil {
		llog.Fatal("初始化 ReAct 工具失败", err)
		return nil
	}
	allTools := append(reactTools, skillTools...)
	skillInjector := func(prompt string) string { return prompt }
	if skillsMiddleware != nil {
		skillInjector = skillsMiddleware.InjectPrompt
	}
	unknownHandler := reactbiz.BuildUnknownToolHandler(allTools, reactLog)
	reactAgent, err := reactagent.NewAgent(context.Background(), &reactagent.AgentConfig{
		ToolCallingModel: reactModel,
		ToolsConfig: compose.ToolsNodeConfig{
			Tools:               allTools,
			UnknownToolsHandler: unknownHandler,
			ToolCallMiddlewares: []compose.ToolMiddleware{reactbiz.TraceToolMiddleware(reactLog)},
		},
		ToolReturnDirectly: map[string]struct{}{reactbiz.ToolFinalResponse: {}},
	})
	if err != nil {
		llog.Fatal("初始化 ReAct agent 失败", err)
		return nil
	}

	chatFlow, err := flow.NewChatFlow(flowtypes.Dependencies{
		ReActAgent:          reactAgent,
		ReActTemplate:       reactTemplate,
		SkillPromptInjector: skillInjector,
		Memory:              memoryManager,
		Frequency:           frequencyManager,
		ReActLog:            reactLog,
	})
	if err != nil {
		llog.Fatal("初始化聊天编排失败", err)
		return nil
	}

	return &ChatEngine{
		ReplyTable: reply,
		memory:     memoryManager,
		frequency:  frequencyManager,
		hooks:      hookRunner,
		flow:       chatFlow,
	}
}

func (c *ChatEngine) ChatWithLanMei(nickname string, input string, ID string, groupId string, must bool) string {
	return c.ChatWithLanMeiWithIntervention(nickname, input, ID, groupId, must, nil)
}

func (c *ChatEngine) ChatWithLanMeiWithIntervention(nickname string, input string, ID string, groupId string, must bool, scores *flowtypes.InterventionScores) string {
	if c == nil || c.flow == nil {
		return ""
	}
	ctx := context.Background()
	reply, err := c.flow.Run(ctx, flowtypes.Request{
		Nickname:     nickname,
		Input:        input,
		UserID:       ID,
		GroupID:      groupId,
		Must:         must,
		Intervention: scores,
	})
	if err != nil {
		llog.Error("chat flow error: %v", err)
		return ""
	}
	return reply
}

func (c *ChatEngine) Shutdown() {
	if c == nil {
		return
	}
	if c.memory != nil {
		c.memory.FlushAll()
	}
}

func mustLoadNodeConfig(name string) llmmodel.NodeConfig {
	cfg := llmmodel.LoadNodeConfig(name)
	if cfg.Provider == "" {
		llog.Fatalf("LLM 节点 %s 缺少 Type 配置", name)
	}
	return cfg
}
