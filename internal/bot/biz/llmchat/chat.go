package llmchat

import (
	"LanMei/internal/bot/biz/dao"
	"LanMei/internal/bot/biz/llmchat/analysis"
	"LanMei/internal/bot/biz/llmchat/flow"
	"LanMei/internal/bot/biz/llmchat/hooks"
	"LanMei/internal/bot/biz/llmchat/memory"
	llmmodel "LanMei/internal/bot/biz/llmchat/model"
	"LanMei/internal/bot/config"
	"LanMei/internal/bot/utils/feishu"
	"LanMei/internal/bot/utils/llog"
	"LanMei/internal/bot/utils/rerank"
	"LanMei/internal/bot/utils/websearch"
	"context"
	"time"
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
	judgeConfig := mustLoadNodeConfig("Judge")
	plannerConfig := mustLoadNodeConfig("Planner")
	analysisConfig := mustLoadNodeConfig("Analysis")
	memoryConfig := mustLoadNodeConfig("Memory")
	searchFormatConfig := mustLoadNodeConfig("SearchFormat")

	chatModel, err := llmmodel.NewChatModel(chatConfig)
	if err != nil {
		llog.Fatal("初始化大模型", err)
		return nil
	}
	plannerModel, err := llmmodel.NewToolCallingChatModel(plannerConfig, buildPlanTool())
	if err != nil {
		llog.Fatal("初始化 planner 工具失败", err)
		return nil
	}
	judgeModel, err := llmmodel.NewToolCallingChatModel(judgeConfig, buildJudgeTool())
	if err != nil {
		llog.Fatal("初始化 judge 模型", err)
		return nil
	}
	analysisModel, err := llmmodel.NewToolCallingChatModel(analysisConfig, analysis.BuildTool())
	if err != nil {
		llog.Fatal("初始化 input 分析工具失败", err)
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
	searchModel, err := llmmodel.NewChatModel(searchFormatConfig)
	if err != nil {
		llog.Fatal("初始化 search_format 模型失败", err)
		return nil
	}

	searchTemplate := buildSearchFormatTemplate()
	template := buildChatTemplate()
	planTemplate := buildPlanTemplate()
	judgeTemplate := buildJudgeTemplate()
	hookRunner := hooks.NewRunner(hooks.NewDurationLogger())
	chatHookInfo := hooks.CallInfo{Node: "chat", Model: chatConfig.Model}
	judgeHookInfo := hooks.CallInfo{Node: "judge", Model: judgeConfig.Model}
	planHookInfo := hooks.CallInfo{Node: "planner", Model: plannerConfig.Model}
	analysisHookInfo := hooks.CallInfo{Node: "analysis", Model: analysisConfig.Model}
	factHookInfo := hooks.CallInfo{Node: "fact_extract", Model: memoryConfig.Model}
	factUpdateHookInfo := hooks.CallInfo{Node: "fact_update", Model: memoryConfig.Model}
	profileHookInfo := hooks.CallInfo{Node: "profile", Model: memoryConfig.Model}
	searchHookInfo := hooks.CallInfo{Node: "search_format", Model: searchFormatConfig.Model}

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
	inputAnalyzer := analysis.NewInputAnalyzer(analysisModel, hookRunner, analysisHookInfo)
	searcher := websearch.NewClient()
	frequencyManager := NewFrequencyControlManager()

	chatFlow, err := flow.NewChatFlow(flow.Dependencies{
		ChatModel:      chatModel,
		JudgeModel:     judgeModel,
		PlannerModel:   plannerModel,
		SearchModel:    searchModel,
		Template:       template,
		JudgeTemplate:  judgeTemplate,
		PlanTemplate:   planTemplate,
		SearchTemplate: searchTemplate,
		InputAnalyzer:  inputAnalyzer,
		Memory:         memoryManager,
		Reranker:       reranker,
		Searcher:       searcher,
		Frequency:      frequencyManager,
		Hooks:          hookRunner,
		HookInfos: flow.HookInfos{
			Chat:   chatHookInfo,
			Judge:  judgeHookInfo,
			Plan:   planHookInfo,
			Search: searchHookInfo,
		},
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
	if c == nil || c.flow == nil {
		return ""
	}
	ctx := context.Background()
	reply, err := c.flow.Run(ctx, flow.Request{
		Nickname: nickname,
		Input:    input,
		UserID:   ID,
		GroupID:  groupId,
		Must:     must,
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
