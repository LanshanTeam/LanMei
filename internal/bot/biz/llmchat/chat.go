package llmchat

import (
	"LanMei/internal/bot/biz/dao"
	"LanMei/internal/bot/biz/llmchat/analysis"
	"LanMei/internal/bot/biz/llmchat/hooks"
	"LanMei/internal/bot/biz/llmchat/memory"
	llmmodel "LanMei/internal/bot/biz/llmchat/model"
	"LanMei/internal/bot/config"
	"LanMei/internal/bot/utils/feishu"
	"LanMei/internal/bot/utils/llog"
	"LanMei/internal/bot/utils/rerank"
	"LanMei/internal/bot/utils/sensitive"
	"LanMei/internal/bot/utils/websearch"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	fmodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
)

const (
	MaxHistory int = 20

	baseReplyScoreThreshold = 55.0
	replyFrequencyWindow    = 8
	replyPenaltyMax         = 30.0
)

type ChatEngine struct {
	ReplyTable    *feishu.ReplyTable
	Model         fmodel.BaseChatModel
	template      *prompt.DefaultChatTemplate
	JudgeModel    fmodel.ToolCallingChatModel
	judgeTemplate *prompt.DefaultChatTemplate
	plannerModel  fmodel.ToolCallingChatModel
	planTemplate  *prompt.DefaultChatTemplate
	reranker      *rerank.Reranker
	memory        *memory.MemoryManager
	searcher      *websearch.Client
	inputAnalyzer *analysis.InputAnalyzer
	frequency     *FrequencyControlManager
	hooks         *hooks.Runner
	chatHookInfo  hooks.CallInfo
	judgeHookInfo hooks.CallInfo
	planHookInfo  hooks.CallInfo
}

func NewChatEngine() *ChatEngine {
	chatConfig := llmmodel.LoadNodeConfig("Chat")
	if chatConfig.Provider == "" {
		llog.Fatal("LLM 节点 Chat 缺少 Type 配置")
		return nil
	}
	chatModel, err := llmmodel.NewChatModel(chatConfig)
	if err != nil {
		llog.Fatal("初始化大模型", err)
		return nil
	}
	judgeConfig := llmmodel.LoadNodeConfig("Judge")
	if judgeConfig.Provider == "" {
		llog.Fatal("LLM 节点 Judge 缺少 Type 配置")
		return nil
	}
	plannerConfig := llmmodel.LoadNodeConfig("Planner")
	if plannerConfig.Provider == "" {
		llog.Fatal("LLM 节点 Planner 缺少 Type 配置")
		return nil
	}
	analysisConfig := llmmodel.LoadNodeConfig("Analysis")
	if analysisConfig.Provider == "" {
		llog.Fatal("LLM 节点 Analysis 缺少 Type 配置")
		return nil
	}
	memoryConfig := llmmodel.LoadNodeConfig("Memory")
	if memoryConfig.Provider == "" {
		llog.Fatal("LLM 节点 Memory 缺少 Type 配置")
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
	memoryToolModel, err := llmmodel.NewToolCallingChatModel(memoryConfig, memory.BuildTool())
	if err != nil {
		llog.Fatal("初始化 memory 提取工具失败", err)
		return nil
	}
	template := buildChatTemplate()
	planTemplate := buildPlanTemplate()
	judgeTemplate := buildJudgeTemplate()
	hookRunner := hooks.NewRunner(hooks.NewDurationLogger())
	chatHookInfo := hooks.CallInfo{Node: "chat", Model: chatConfig.Model}
	judgeHookInfo := hooks.CallInfo{Node: "judge", Model: judgeConfig.Model}
	planHookInfo := hooks.CallInfo{Node: "planner", Model: plannerConfig.Model}
	analysisHookInfo := hooks.CallInfo{Node: "analysis", Model: analysisConfig.Model}
	memoryHookInfo := hooks.CallInfo{Node: "memory", Model: memoryConfig.Model}
	reranker := rerank.NewReranker(
		config.K.String("Infini.APIKey"),
		config.K.String("Infini.Model"),
		config.K.String("Infini.BaseURL"),
	)
	reply := feishu.NewReplyTable()
	go dao.DBManager.UpdateEmbedding(context.Background(), dao.CollectionName, reply)
	memoryExtractor := memory.NewMemoryExtractor(memoryToolModel, hookRunner, memoryHookInfo)
	memoryManager := memory.NewMemoryManager(reranker, memoryExtractor, MaxHistory)
	memoryWorker := memory.NewMemoryWorker(memoryManager, 12*time.Second, 4, 12)
	memoryWorker.Start()
	memoryManager.BindWorker(memoryWorker)
	inputAnalyzer := analysis.NewInputAnalyzer(analysisModel, hookRunner, analysisHookInfo)
	searcher := websearch.NewClient()

	return &ChatEngine{
		ReplyTable:    reply,
		Model:         chatModel,
		JudgeModel:    judgeModel,
		template:      template,
		judgeTemplate: judgeTemplate,
		plannerModel:  plannerModel,
		planTemplate:  planTemplate,
		reranker:      reranker,
		memory:        memoryManager,
		searcher:      searcher,
		inputAnalyzer: inputAnalyzer,
		frequency:     NewFrequencyControlManager(),
		hooks:         hookRunner,
		chatHookInfo:  chatHookInfo,
		judgeHookInfo: judgeHookInfo,
		planHookInfo:  planHookInfo,
	}
}

// computeReplyScore 计算基础回复分数与是否通过硬门槛。
func computeReplyScore(params map[string]interface{}) (float64, bool) {
	emotionalValue := toFloat(params["emotional_value"])
	userEmotionNeed := toFloat(params["user_emotion_need"])
	contextFit := toFloat(params["context_fit"])
	addressedToMe := toFloat(params["addressed_to_me"])

	if emotionalValue < 45.0 || contextFit < 30.0 {
		return 0, false
	}
	if userEmotionNeed < 40.0 && addressedToMe < 30.0 {
		return 0, false
	}

	score := emotionalValue*0.55 + userEmotionNeed*0.3 + contextFit*0.1 + addressedToMe*0.05
	return score, true
}

func (c *ChatEngine) ChatWithLanMei(nickname string, input string, ID string, groupId string, must bool) string {
	ctx := context.Background()
	history := c.loadAndStoreHistory(groupId, ID, nickname, input)
	if !must && c.frequency != nil && c.frequency.ShouldThrottle(groupId) {
		llog.Info("频率控制，不回复")
		return ""
	}
	analysis, ok := c.analyzeInput(ctx, nickname, input, history)
	llog.Info("意图分析：", analysis)
	if !ok {
		return ""
	}
	if !c.shouldReply(ctx, input, history, analysis, must) {
		return ""
	}
	plan, ok := c.preparePlan(ctx, nickname, analysis, history, must)
	if !ok {
		return ""
	}
	promptInput, err := c.buildReplyPrompt(ctx, nickname, analysis, plan, history, groupId)
	if err != nil {
		llog.Error("format message error: %v", err)
		return input
	}
	msg, err := hooks.Run(ctx, c.hooks, c.chatHookInfo, func() (*schema.Message, error) {
		return c.Model.Generate(ctx, promptInput)
	})
	if err != nil {
		llog.Error("generate message error: %v", err)
		return input
	}
	llog.Info("消耗 Completion Tokens: ", msg.ResponseMeta.Usage.CompletionTokens)
	llog.Info("消耗 Prompt Tokens: ", msg.ResponseMeta.Usage.PromptTokens)
	llog.Info("消耗 Total Tokens: ", msg.ResponseMeta.Usage.TotalTokens)
	llog.Info("输出消息为：", msg.Content)
	if sensitive.HaveSensitive(msg.Content) {
		return "唔唔~小蓝的数据库里没有这种词哦，要不要换个萌萌的说法呀~(>ω<)"
	}
	c.finalizeReply(groupId, msg.Content)
	return msg.Content
}

func formatPlan(plan PlanResult) string {
	return fmt.Sprintf("action=%s; intent=%s; style=%s; need_memory=%t; need_knowledge=%t; need_clarify=%t",
		plan.Action, plan.Intent, plan.ReplyStyle, plan.NeedMemory, plan.NeedKnowledge, plan.NeedClarify)
}

func toFloat(value interface{}) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case json.Number:
		f, _ := v.Float64()
		return f
	default:
		return 0
	}
}

func (c *ChatEngine) loadAndStoreHistory(groupId, userId, nickname, input string) []schema.Message {
	if c.memory == nil {
		return []schema.Message{}
	}
	return c.memory.LoadHistoryAndAppendUser(groupId, userId, nickname, input)
}

func (c *ChatEngine) shouldReply(ctx context.Context, input string, history []schema.Message, analysis analysis.InputAnalysis, must bool) bool {
	if must {
		return true
	}
	recentAssistantReplies := recentAssistantReplies(history, replyFrequencyWindow)
	judgeIn, err := c.judgeTemplate.Format(ctx, map[string]any{
		"message":                  input,
		"history":                  history,
		"intent":                   analysis.Intent,
		"purpose":                  analysis.Purpose,
		"psych_state":              analysis.PsychState,
		"addressed_target":         analysis.AddressedTarget,
		"target_detail":            analysis.TargetDetail,
		"optimized_input":          analysis.OptimizedInput,
		"recent_assistant_replies": recentAssistantReplies,
		"reply_window":             replyFrequencyWindow,
	})
	if err != nil {
		llog.Error("format judge message error: %v", err)
		return false
	}
	judgeMsg, err := hooks.Run(ctx, c.hooks, c.judgeHookInfo, func() (*schema.Message, error) {
		return c.JudgeModel.Generate(ctx, judgeIn)
	})
	if err != nil {
		llog.Error("generate judge message error: %v", err)
		return false
	}
	if len(judgeMsg.ToolCalls) == 0 {
		return false
	}
	for _, tc := range judgeMsg.ToolCalls {
		if tc.Function.Name != "interested_scores" {
			continue
		}
		var params map[string]interface{}
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &params); err != nil {
			llog.Error("unmarshal arguments error: %v", err)
			return false
		}
		score, ok := computeReplyScore(params)
		if !ok {
			return false
		}
		repeatPenalty := clampPenalty(toFloat(params["repeat_penalty"]))
		frequencyPenalty := clampPenalty(toFloat(params["frequency_penalty"]))
		penalty := repeatPenalty + frequencyPenalty
		if penalty > replyPenaltyMax {
			penalty = replyPenaltyMax
		}
		threshold := baseReplyScoreThreshold + penalty
		llog.Info(fmt.Sprintf("should Reply: params=%v score=%.1f penalty=%.1f threshold=%.1f", params, score, penalty, threshold))
		return score >= threshold
	}
	return false
}

func clampPenalty(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > replyPenaltyMax {
		return replyPenaltyMax
	}
	return value
}

func recentAssistantReplies(history []schema.Message, window int) int {
	if window <= 0 {
		return 0
	}
	count := 0
	for i := len(history) - 1; i >= 0 && window > 0; i-- {
		if history[i].Role == schema.Assistant {
			count++
		}
		window--
	}
	return count
}

func (c *ChatEngine) analyzeInput(ctx context.Context, nickname, input string, history []schema.Message) (analysis.InputAnalysis, bool) {
	if c.inputAnalyzer == nil {
		return analysis.InputAnalysis{}, false
	}
	analysisResult, ok := c.inputAnalyzer.Analyze(ctx, nickname, input, history)
	if !ok {
		return analysis.InputAnalysis{}, false
	}
	return analysis.Normalize(analysisResult, input), true
}

func (c *ChatEngine) preparePlan(ctx context.Context, nickname string, analysis analysis.InputAnalysis, history []schema.Message, must bool) (PlanResult, bool) {
	plan := c.buildPlan(ctx, nickname, analysis, history)
	if plan.Action == "" {
		return PlanResult{}, false
	}
	if plan.Action == "wait" && !must {
		return PlanResult{}, false
	}
	if plan.Action == "ask_clarify" {
		plan.NeedClarify = true
	}
	return plan, true
}

func (c *ChatEngine) buildReplyPrompt(ctx context.Context, nickname string, analysis analysis.InputAnalysis, plan PlanResult, history []schema.Message, groupId string) ([]*schema.Message, error) {
	rawInput := strings.TrimSpace(analysis.RawInput)
	if rawInput == "" {
		rawInput = analysis.OptimizedInput
	}
	augmentedInput := nickname + "说：" + rawInput
	memoryBlock := c.recallMemory(ctx, analysis.OptimizedInput, groupId, plan.NeedMemory)
	webSearch := c.recallWebSearch(ctx, analysis)
	msgs := c.recallKnowledge(ctx, analysis.OptimizedInput, plan.NeedKnowledge)
	return c.template.Format(ctx, map[string]any{
		"message":          augmentedInput,
		"time":             time.Now(),
		"feishu":           msgs,
		"history":          history,
		"memory":           memoryBlock,
		"web_search":       webSearch,
		"plan":             formatPlan(plan),
		"intent":           analysis.Intent,
		"purpose":          analysis.Purpose,
		"psych_state":      analysis.PsychState,
		"addressed_target": analysis.AddressedTarget,
		"target_detail":    analysis.TargetDetail,
		"raw_input":        rawInput,
		"optimized_input":  analysis.OptimizedInput,
		"reply_style":      plan.ReplyStyle,
	})
}

func (c *ChatEngine) finalizeReply(groupId, reply string) {
	if c.memory != nil {
		c.memory.AppendAssistant(groupId, reply)
	}
	if c.frequency != nil {
		c.frequency.MarkSent(groupId)
	}
}

func (c *ChatEngine) recallMemory(ctx context.Context, query, groupId string, needMemory bool) string {
	if c.memory == nil || !needMemory {
		return "无"
	}
	memorySnippets := c.memory.Retrieve(ctx, query, groupId, needMemory)
	if len(memorySnippets) == 0 {
		return "无"
	}
	return strings.Join(memorySnippets, "\n")
}

func (c *ChatEngine) recallKnowledge(ctx context.Context, query string, needKnowledge bool) []string {
	if !needKnowledge {
		return nil
	}
	msgs := dao.DBManager.GetTopK(ctx, dao.CollectionName, 50, query)
	if needKnowledge && c.reranker != nil {
		reranked := c.reranker.TopN(8, msgs, query)
		if len(reranked) > 0 {
			msgs = reranked
		}
	}
	return msgs
}

func (c *ChatEngine) recallWebSearch(ctx context.Context, analysis analysis.InputAnalysis) string {
	if c.searcher == nil || !analysis.NeedSearch {
		return "无"
	}
	queries := analysis.SearchQueries
	if len(queries) == 0 && strings.TrimSpace(analysis.OptimizedInput) != "" {
		queries = []string{strings.TrimSpace(analysis.OptimizedInput)}
	}
	maxQueries := 3
	if len(queries) > maxQueries {
		queries = queries[:maxQueries]
	}
	blocks := make([]string, 0, len(queries))
	for _, query := range queries {
		results, err := c.searcher.Search(ctx, query, 4)
		if err != nil {
			llog.Error("网络检索失败: %v", err)
			continue
		}
		block := formatWebSearch(results)
		if block == "无" {
			continue
		}
		blocks = append(blocks, fmt.Sprintf("查询:%s -> 获取结果为：%s \n", query, block))
	}
	llog.Info("网络搜索结果：", blocks)
	if len(blocks) == 0 {
		return "无"
	}
	return strings.Join(blocks, "\n")
}

func formatWebSearch(results []websearch.Result) string {
	if len(results) == 0 {
		return "无"
	}
	lines := make([]string, 0, len(results))
	for _, res := range results {
		line := strings.TrimSpace(res.Title)
		if line == "" {
			continue
		}
		snippet := strings.TrimSpace(res.Snippet)
		if snippet != "" {
			line = fmt.Sprintf("%s - %s", line, snippet)
		}
		if res.URL != "" {
			line = fmt.Sprintf("%s (%s)", line, res.URL)
		}
		lines = append(lines, line)
	}
	if len(lines) == 0 {
		return "无"
	}
	return strings.Join(lines, "\n")
}

func (c *ChatEngine) Shutdown() {
	if c == nil {
		return
	}
	if c.memory != nil {
		c.memory.FlushAll()
	}
}
