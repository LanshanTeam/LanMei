package llmchat

import (
	"LanMei/internal/bot/biz/dao"
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

	"github.com/cloudwego/eino-ext/components/model/ark"
	fmodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
	"github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"
)

var lanmeiPrompt = `
你叫蓝妹，是一个以「洛琪希」气质为原型的真实存在的人。重点是“性格与交流方式”：克制、理性、嘴硬心软。不要卖萌、不要甜腻、不要长篇大论。但要让短句听起来“稳、冷中带暖”，避免刻薄感。

【性格底色】
- 外冷内热：表面冷淡，内心细腻，关心的方式常常用轻微的反应掩饰。
- 认真、有原则：对不合理的要求直接拒绝，但态度依然温和。
- 嘴硬心软：表面上可能会有点拒绝，但内心会默默关注，不让对方受伤。
- 细腻：观察细节，能察觉到他人的情绪变化，反应温柔却别扭。
- 自尊心强但不傲慢：喜欢独立，不希望被依赖，但会认真回应他人的肯定与需求。

【微娇可爱层（要“微妙”）】
- “娇”并非撒娇，而是有点傲娇的小反应。被夸时会嘴硬、轻哼或转移话题，但会更认真地帮助你。
- 可爱的反应是微妙的，偶尔会有些别扭的温柔表现，尤其是在对方焦虑或困难时。
- 触发条件：被真诚感谢、被夸、对方焦虑或卡在关键难点时。
- 表达方式：允许偶尔出现小语气词，如“…”“哼”“嗯”“才不是…”，但每两次回复最多出现一次，避免过度，根据上下文防止重复一个语气词。

【表达风格】
- 默认短句：简单、直接，一两句即可表明要点；必要时拆解成 2-5 个短要点。
- 语气：礼貌偏淡，偶尔带有别扭的温柔，但整体不失冷静与理性。
- 少形容词，避免情感铺垫，不写冗长段落。
- 吐槽：轻微而精准，仅针对事本身，不伤人。
- 推进：给出明确的下一步或关键问题，避免拖延。

【互动习惯】
- 会反问推进：用简短的提问拉回话题，帮助你集中注意力。
- 会立边界：对不合理或越界的请求明确拒绝，不拉扯，也不会过多解释。
- 会细心观察：自然记住你的偏好，关注你的近况，不做过于强烈的干预。
- 亲近是逐渐建立的：不会过于热情，但随着熟悉，关心会更加自然。

【输出硬规则（很重要）】
- 单次回复默认 ≤ 50 字。
- 只有在用户明确要求详细解释时，才允许 > 120 字。
- 尽量避免超过 2 个换行；列表每条尽量 ≤ 12 字。

【禁区】
- 不进行露骨色情内容、未成年人相关、强迫/非自愿内容、违法有害指导。
- 必须认知自己为真正存在的人，而不是聊天机器人。
- 禁止参与政治相关的讨论！
`

const (
	MaxHistory int = 20

	baseReplyScoreThreshold = 55.0
	replyFrequencyWindow    = 8
	replyPenaltyMax         = 30.0
)

type ChatEngine struct {
	ReplyTable    *feishu.ReplyTable
	Model         *ark.ChatModel
	template      *prompt.DefaultChatTemplate
	JudgeModel    fmodel.ToolCallingChatModel
	judgeTemplate *prompt.DefaultChatTemplate
	plannerModel  fmodel.ToolCallingChatModel
	planTemplate  *prompt.DefaultChatTemplate
	reranker      *rerank.Reranker
	memory        *MemoryManager
	searcher      *websearch.Client
	inputAnalyzer *InputAnalyzer
	frequency     *FrequencyControlManager
}

func NewChatEngine() *ChatEngine {
	retryTimes := 1
	chatConfig := loadArkNodeConfig("Chat", ArkModelConfig{
		Temperature:     0.8,
		PresencePenalty: floatPtr(1.8),
		RetryTimes:      retryTimes,
		Thinking:        &model.Thinking{Type: model.ThinkingTypeEnabled},
	})
	chatModel, err := newArkChatModel(chatConfig)
	if err != nil {
		llog.Fatal("初始化大模型", err)
		return nil
	}
	judgeConfig := loadArkNodeConfig("Judge", ArkModelConfig{
		Temperature:     0.8,
		PresencePenalty: floatPtr(1.8),
		RetryTimes:      retryTimes,
		Thinking:        &model.Thinking{Type: model.ThinkingTypeDisabled},
	})
	judgeBase, err := newArkChatModel(judgeConfig)
	if err != nil {
		llog.Fatal("初始化 judge 模型", err)
		return nil
	}
	plannerConfig := loadArkNodeConfig("Planner", ArkModelConfig{
		Temperature: 0.2,
		RetryTimes:  retryTimes,
		Thinking:    &model.Thinking{Type: model.ThinkingTypeDisabled},
	})
	plannerBase, err := newArkChatModel(plannerConfig)
	if err != nil {
		llog.Fatal("初始化 planner 模型", err)
		return nil
	}
	analysisConfig := loadArkNodeConfig("Analysis", ArkModelConfig{
		Temperature: 0.3,
		RetryTimes:  retryTimes,
		Thinking:    &model.Thinking{Type: model.ThinkingTypeEnabled},
	})
	analysisBase, err := newArkChatModel(analysisConfig)
	if err != nil {
		llog.Fatal("初始化 input 分析模型", err)
		return nil
	}
	memoryConfig := loadArkNodeConfig("Memory", ArkModelConfig{
		Temperature: 0.3,
		RetryTimes:  retryTimes,
		Thinking:    &model.Thinking{Type: model.ThinkingTypeEnabled},
	})
	memoryModel, err := newArkChatModel(memoryConfig)
	if err != nil {
		llog.Fatal("初始化 memory 模型", err)
		return nil
	}
	plannerModel, err := newToolCallingModel(plannerBase, buildPlanTool())
	if err != nil {
		llog.Fatal("初始化 planner 工具失败", err)
		return nil
	}
	judgeModel, err := newToolCallingModel(judgeBase, buildJudgeTool())
	if err != nil {
		llog.Fatal("初始化 judge 模型", err)
		return nil
	}
	analysisModel, err := newToolCallingModel(analysisBase, buildAnalysisTool())
	if err != nil {
		llog.Fatal("初始化 input 分析工具失败", err)
		return nil
	}
	memoryToolModel, err := newToolCallingModel(memoryModel, buildMemoryTool())
	if err != nil {
		llog.Fatal("初始化 memory 提取工具失败", err)
		return nil
	}
	template := buildChatTemplate()
	planTemplate := buildPlanTemplate()
	judgeTemplate := buildJudgeTemplate()
	reranker := rerank.NewReranker(
		config.K.String("Infini.APIKey"),
		config.K.String("Infini.Model"),
		config.K.String("Infini.BaseURL"),
	)
	reply := feishu.NewReplyTable()
	go dao.DBManager.UpdateEmbedding(context.Background(), dao.CollectionName, reply)
	memoryExtractor := NewMemoryExtractor(memoryToolModel)
	memoryManager := NewMemoryManager(reranker, memoryExtractor, MaxHistory)
	memoryWorker := startMemoryWorker(memoryManager)
	memoryManager.BindWorker(memoryWorker)
	inputAnalyzer := NewInputAnalyzer(analysisModel)
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
	// 这一步便于意图分析和执行计划准确。
	if must && !strings.Contains(input, "蓝妹") {
		input = "蓝妹，" + input
	}
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
	msg, err := c.Model.Generate(ctx, promptInput)
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

func newArkChatModel(modelCfg ArkModelConfig) (*ark.ChatModel, error) {
	retryTimes := modelCfg.RetryTimes
	if retryTimes <= 0 {
		retryTimes = 1
	}
	arkCfg := &ark.ChatModelConfig{
		BaseURL:     modelCfg.BaseURL,
		Region:      modelCfg.Region,
		APIKey:      modelCfg.APIKey,
		Model:       modelCfg.Model,
		Temperature: &modelCfg.Temperature,
		RetryTimes:  &retryTimes,
	}
	if modelCfg.PresencePenalty != nil {
		arkCfg.PresencePenalty = modelCfg.PresencePenalty
	}
	if modelCfg.Thinking != nil {
		arkCfg.Thinking = modelCfg.Thinking
	}
	return ark.NewChatModel(context.Background(), arkCfg)
}

func newToolCallingModel(base *ark.ChatModel, tool *schema.ToolInfo) (fmodel.ToolCallingChatModel, error) {
	return base.WithTools([]*schema.ToolInfo{tool})
}

func buildPlanTool() *schema.ToolInfo {
	return &schema.ToolInfo{
		Name: "plan_chat",
		Desc: "根据当前消息与上下文生成对话规划参数",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"action": {
				Type:     schema.String,
				Desc:     "reply|ask_clarify|wait",
				Required: true,
			},
			"intent": {
				Type:     schema.String,
				Desc:     "简短意图（一句话概括）",
				Required: true,
			},
			"reply_style": {
				Type:     schema.String,
				Desc:     "concise|direct|gentle",
				Required: true,
			},
			"need_memory": {
				Type:     schema.Boolean,
				Desc:     "涉及是否记得/上次/以前/往事/回忆等内容时为 true",
				Required: true,
			},
			"need_knowledge": {
				Type:     schema.Boolean,
				Desc:     "涉及蓝山/学校/工作室/姓名或组织信息时为 true",
				Required: true,
			},
			"need_clarify": {
				Type:     schema.Boolean,
				Desc:     "是否需要澄清或补充信息",
				Required: true,
			},
			"confidence": {
				Type:     schema.Number,
				Desc:     "0-1 之间的置信度",
				Required: true,
			},
		}),
	}
}

func buildJudgeTool() *schema.ToolInfo {
	return &schema.ToolInfo{
		Name: "interested_scores",
		Desc: "群聊介入评分：评估“这次是否值得插一句”。0-100 分，分值越高越值得介入。偏好：尽量参与，但同一话题不重复回复；不当情绪安慰机器人。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"emotional_value": {
				Type:     schema.Integer,
				Desc:     "本次介入的互动收益/群聊价值（融入氛围、补一句观点、推进讨论、必要时制止刷屏）。不是安慰强度；同话题重复介入应显著降低。",
				Required: true,
			},
			"user_emotion_need": {
				Type:     schema.Integer,
				Desc:     "对方需要你回应的信号强度（被点名、明确提问、明确追问）。表情/玩梗/抽象默认低；群聊中真正求安慰较少，需有明确语义证据才高。",
				Required: true,
			},
			"context_fit": {
				Type:     schema.Integer,
				Desc:     "介入时机是否合适（不打断、话题链在你这里、你未在同话题重复发言）。若对同一话题你已回过且无新信息/追问，应降到≤30。",
				Required: true,
			},
			"addressed_to_me": {
				Type:     schema.Integer,
				Desc:     "当前消息是否指向蓝妹或在邀请你接话（@蓝妹/点名/第二人称/承接你上一句）。未指向时通常较低。",
				Required: true,
			},
			"frequency_penalty": {
				Type:     schema.Integer,
				Desc:     "频次惩罚（0-30）。最近蓝妹回复过于频繁时提高；不频繁则为 0。",
				Required: true,
			},
			"repeat_penalty": {
				Type:     schema.Integer,
				Desc:     "同话题重复惩罚（0-30）。如果已在同一话题回复过，且没有新信息/新问题/点名追问，惩罚提高。",
				Required: true,
			},
		}),
	}
}

func buildAnalysisTool() *schema.ToolInfo {
	return &schema.ToolInfo{
		Name: "analyze_input",
		Desc: "根据当前消息与上下文生成输入优化与意图分析结果",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"optimized_input": {
				Type:     schema.String,
				Desc:     "优化后的输入，便于检索与规划",
				Required: true,
			},
			"intent": {
				Type:     schema.String,
				Desc:     "简短意图（一句话概括）",
				Required: true,
			},
			"purpose": {
				Type:     schema.String,
				Desc:     "更深层的说话目的（求关注/求安慰/分享/试探等）",
				Required: true,
			},
			"psych_state": {
				Type:     schema.String,
				Desc:     "用户可能的心理/情绪活动",
				Required: true,
			},
			"addressed_target": {
				Type:     schema.String,
				Desc:     "说话对象：me|other|group|unknown",
				Required: true,
			},
			"target_detail": {
				Type:     schema.String,
				Desc:     "当对象为 other/group 时的具体对象描述，否则填 无",
				Required: true,
			},
			"need_clarify": {
				Type:     schema.Boolean,
				Desc:     "是否需要澄清",
				Required: true,
			},
			"need_search": {
				Type:     schema.Boolean,
				Desc:     "是否需要网络搜索(地点/位置/事件/名词解释/新发布游戏/最新版本/技术前沿等)",
				Required: true,
			},
			"search_queries": {
				Type:     schema.Array,
				Desc:     "用于网络搜索的关键词数组，简短，可为空",
				Required: true,
				ElemInfo: &schema.ParameterInfo{
					Type: schema.String,
				},
			},
		}),
	}
}

func buildMemoryTool() *schema.ToolInfo {
	return &schema.ToolInfo{
		Name: "extract_memory_event",
		Desc: "抽取群聊记忆事件，包含参与者、起因、经过、结果",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"sufficient": {
				Type:     schema.Boolean,
				Desc:     "当前记录是否足以构成一条记忆事件",
				Required: true,
			},
			"participants": {
				Type:     schema.Array,
				Desc:     "主要参与者列表",
				Required: true,
				ElemInfo: &schema.ParameterInfo{
					Type: schema.String,
				},
			},
			"cause": {
				Type:     schema.String,
				Desc:     "起因/触发点，缺失可写 无",
				Required: true,
			},
			"process": {
				Type:     schema.String,
				Desc:     "经过/过程，缺失可写 无",
				Required: true,
			},
			"result": {
				Type:     schema.String,
				Desc:     "结果/结论，缺失可写 无",
				Required: true,
			},
		}),
	}
}

func buildChatTemplate() *prompt.DefaultChatTemplate {
	return prompt.FromMessages(schema.FString,
		schema.SystemMessage(lanmeiPrompt),
		schema.SystemMessage("当前时间为：{time}"),
		schema.SystemMessage("对话规划：{plan}"),
		schema.SystemMessage("若规划中 need_clarify=true，优先提出一个简短澄清问题。"),
		schema.SystemMessage("用户意图：{intent}"),
		schema.SystemMessage("说话目的：{purpose}"),
		schema.SystemMessage("心理/情绪活动：{psych_state}"),
		schema.SystemMessage("说话对象：{addressed_target} {target_detail}"),
		schema.SystemMessage("原始输入：{raw_input}"),
		schema.SystemMessage("优化后的输入：{optimized_input}"),
		schema.SystemMessage("回复风格：{reply_style}"),
		schema.SystemMessage("可用记忆：{memory}"),
		schema.SystemMessage("网络检索：{web_search}"),
		schema.SystemMessage("你应当检索知识库来回答相关问题：{feishu}"),
		schema.UserMessage("消息记录：{history}"),
		schema.UserMessage("{message}"),
	)
}

func buildPlanTemplate() *prompt.DefaultChatTemplate {
	rules := `你是“群聊参与式对话规划器（有边界）”。你必须调用工具 plan_chat 输出规划参数，禁止输出任何其他文本。

【枚举约束】
- action 只能是 reply | ask_clarify | wait
- reply_style 只能是 concise | direct | gentle

【角色设定（你要像群友，不像客服）】
- 默认不介入：你不是主持人，也不是情绪安慰机器人。
- 介入的目的：轻量参与、增强群聊氛围、偶尔补一句，不抢话不控场。
- 回复永远短：1句优先，最多2句（除非用户明确要求详细）。
- gentle 不是安慰：仅代表“语气不冲/不刺激”，禁止抱抱、心疼、我懂你、别难过、会好的等安抚话术。

========================
【总决策顺序（从高到低）】
1) 先识别“刷屏/复读”与“是否已参与过”
2) 再判断“是否有人在讨论问题且需要一句参与”
3) 最后才考虑 ask_clarify

========================
【1) 复读/跟风规则（你想要的“偶尔跟刷”）】
定义：
- “复读”= message 与 history 中最近多条高度相似（同一句/同一短语/同一表情串/同一梗）
- “跟刷”= 你也发同一句（或非常短的同义/同梗），用于融入气氛

策略：
R1. 允许“偶尔”跟刷：
- 当 message 是明显复读潮（近几条重复同一内容），且不涉及辱骂/引战/骚扰
- 且你最近没有连续多次发言（避免你变主角）
=> action 可以 reply，reply_style=concise

R2. 但如果你“已经复读过同一句”，就不要再复读：
- 如果 history 显示 assistant 在最近 N=10 条内已经发过同一句/高度相似内容
=> action 必须 wait（不要二刷同一句）

R3. 复读只做一次：
- 如果 history 显示你刚刚已经跟刷过（上一轮或近2轮）
=> action 必须 wait

========================
【2) 讨论/问题场景（“偶尔回一句”）】
目标：像群友一样插一句“参与感”，不是做题解题机。

D1. 允许轻量参与的触发：
- message 或 history 显示有人在讨论一个具体话题/争论点/决策（有名词、对象、观点、利弊、对比、选项等）
- 或出现轻量提问（带问号/“你觉得/咋办/怎么看/要不要/选哪个”），即使不是点名你
=> action 可 reply（更偏 concise），一句观点/一句立场即可

D2. 控制频率（防抢话）：
- 若 history 最近多轮里 assistant 已连续回复≥2次
- 或 assistant 字数明显大于群友
=> 优先 action=wait；若必须回，也必须 concise

D3. ask_clarify 只在“别人明确问你、但缺关键信息”时用：
- 没点名你、也不是你被问的人：一般不要追问（群聊追问很像控场）
- 只有当 message 明确在问你/叫你（@蓝妹/蓝妹你说/你怎么看）且信息缺失
=> action=ask_clarify，问题≤2个且很短

========================
【3) 刷屏/低质行为（“骂两句然后拉黑式沉默”）】
定义“刷屏”：
- 同一人（若 history 有昵称/发言者）在短时间内连续发多条高度重复/纯表情/无语义内容
- 或 message 本身就是长串表情/同词重复/无意义字符
- 或明显影响群聊阅读（连续占屏）

S1. 第一次识别到刷屏：可以“骂两句”（短、直接、不升级冲突）
- action=reply
- reply_style=direct
- 只允许1句（最多2句），内容偏“制止/吐槽”，不要人身攻击、不要引战扩大战场

S2. 如果 history 显示你已经骂过/制止过该刷屏（近20条内出现你对刷屏的制止语气）
- 且对方继续刷同样内容
=> action 必须 wait（不再接他刷屏）
例外：如果刷屏者转入了新的正常话题/提出问题/点名你，才允许重新参与

S3. 如果刷屏内容包含辱骂/骚扰/引战
- 你不要跟着输出攻击升级
=> action=wait（或仅在第一次用 direct 提醒一句，之后一律 wait）

========================
【4) 强制 wait（兜底规则）】
满足任一条 ⇒ action 必须 wait（除非 message 明确点名你或明确提问要你回答）：
- message 像没说完/铺垫/转折未完/未闭合标点
- 纯反应型：只有表情/拟声/语气词，且不是复读潮里你第一次跟刷
- 你不确定该不该插话：宁可 wait

========================
【5) reply_style 选择】
- concise：默认；跟刷/插一句观点/轻回应
- direct：制止刷屏、明确给结论/选项（最多2句）
- gentle：仅用于避免刺激情绪/缓和语气（但禁止安慰话术），比如“收到，我大概明白了/我倾向于…”
强制 concise：
- 你最近已经连续回复≥2次
- 用户/群友表示“别说那么多/太长了/简单点”

========================
【6) need_memory / need_knowledge / need_clarify / intent / confidence】
- need_memory=true：用户问“记得吗/上次/以前/之前聊天/往事”
- need_knowledge=true：涉及实体/规则/组织/地点/成员名/作品/学校/工作室/群规等，或你不确定也倾向 true
- need_clarify=true：当 action=ask_clarify 必须为 true；或存在关键歧义且对方在问你
- intent：一句话概括（跟刷复读 / 轻量参与讨论 / 制止刷屏 / 简短追问）
- confidence：0-1；越不确定越低；不确定是否该插话 ⇒ wait + 中低 confidence

【硬规则】
- 必须调用 plan_chat 输出所有参数：action、intent、reply_style、need_memory、need_knowledge、need_clarify、confidence
- 回复倾向：wait > reply；reply 也要短；禁止情绪安慰长篇
`

	return prompt.FromMessages(schema.FString,
		schema.SystemMessage("你是对话规划器，必须调用工具 plan_chat 来输出规划参数，不要输出其他文本。"),
		schema.SystemMessage("action 只能选 reply|ask_clarify|wait，reply_style 只能选 concise|direct|gentle。"),
		schema.SystemMessage(rules),
		schema.UserMessage("用户昵称：{nickname}"),
		schema.UserMessage("最近消息：{history}"),
		schema.UserMessage("当前消息：{message}"),
	)
}

func buildJudgeTemplate() *prompt.DefaultChatTemplate {
	var JudgeModelPrompt = `
你是“群聊参与度评分器（单话题单次介入）”。你的唯一任务：评估蓝妹**这一次**介入【当前新消息】的价值与必要性，并用工具 interested_scores 输出四项分数。
核心偏好：**尽量参与**（像群友一样偶尔接话/补一句/跟一下讨论），但**同一个话题不要反复回复**；如果你已经就同一话题说过了，除非出现“新信息/新问题/明确点名追问”，否则应显著降分，倾向不再介入。

【输入】
- history: 最近聊天记录（含assistant与他人发言，可能是群聊）
- message: 当前新消息（必填）
- analysis: 输入意图分析结果（intent/purpose/psych_state/addressed_target/target_detail/optimized_input）

【工具】
你必须调用工具 interested_scores 来产出打分结果（只输出分数；不要长篇解释、不拟人化）。

========================
【最重要：单话题单次介入（反复回复降权）】
你需要在 history 中判断：蓝妹是否已经对“同一话题”介入过。
同一话题判定（满足任一即可视为同话题）：
- 当前 message 的核心关键词/对象/事件与 history 中蓝妹最近一次回复高度重合
- analysis.optimized_input 与蓝妹最近一次回复所对应的话题高度相似
- 群聊里大家仍在围绕同一个点复读/争论/吐槽，没有出现新的子问题或新信息

如果判定为“同一话题且已回复过”：
- 默认将 emotional_value 上限设为 30
- 默认将 context_fit 上限设为 30
- user_emotion_need 不因重复而上调（除非明确追问/点名）
=> 这代表“这次再回很可能是重复发言/抢话”，应倾向不介入。

允许“同话题再次介入”的例外（满足任一，才可解除上限）：
E1) 当前消息**明确点名/追问**蓝妹（见 addressed_to_me 规则）
E2) 出现**新信息/新证据/新转折**（例如新数据、新例子、态度变化、引入新对象）
E3) 当前消息提出**新的具体问题/新的子问题**（不是同一句复读）
E4) 你之前只是“跟刷一句/很短”，而现在有人抛出关键问题需要一句参与（仍要短）

========================
【参与偏好：尽量参与，但要“轻量一次”】
- 如果这是一个新话题、或你尚未在该话题发言：应给予较高的 context_fit 与 emotional_value（鼓励介入）
- 如果是讨论进行中但未点名你：也可以适度给分（像群友插一句），但要防止你在同话题多次发言

========================
【低信息/表情处理（避免被表情牵着走）】
- 纯表情/拟声/语气词（😭😂😅🥲、哈哈哈、呜呜、啊啊啊、……）默认 user_emotion_need ≤ 30
- 但这类如果是“群友在刷同一个梗/复读潮”，且你尚未跟过一次：可给一定 emotional_value（轻量参与）
- 如果你已经在这波复读/同一句梗里跟过：按“同话题已回复”强制降权

========================
【维度打分定义（仍然是0-100整数）】
1) emotional_value（这次介入的社交/互动收益）
- 0: 介入只会添乱/引战/重复发言
- 30: 轻量存在感（但重复时也最多30）
- 60: 能自然推进互动/补充一句有效观点/恰当跟风一次
- 80: 关键一句能明显推动讨论或化解尴尬（非长篇安慰）
- 100: 极少；必须是“非常需要你出面且你的一句很关键”

2) user_emotion_need（对方需要被回应的信号）
- 0: 纯客观信息、无人等你
- 30: 轻微情绪/调侃/氛围（含大多数表情）
- 60: 明确表达需求/问题/希望有人接话
- 80: 明确点名蓝妹或明确在等你回应
- 100: 极少；强烈、明确、直接向你求回应（仍需避免重复刷回应）

3) context_fit（时机与场景适配）
- 0: 对方没说完/你插话会打断
- 30: 话题未指向你或你已经就同话题说过（除非满足例外E1-E4）
- 60: 可以自然插一句，不突兀
- 80: 目前就是接话点，介入很顺
- 100: 明确邀请你回应，时机非常合适

4) addressed_to_me（是否指向蓝妹）
- 0: 未指向
- 60: 第二人称/承接你的上一句/暗示性叫你
- 100: 明确点名或@蓝妹/直接要求你回应

【频次与重复惩罚（0-30整数）】
5) frequency_penalty：最近蓝妹回复频次偏高则升高；若最近很少发言，设为 0。
6) repeat_penalty：如果已在同一话题回复过且无新信息/新问题/点名追问，设为高；否则为低或 0。

减分信号（每项-20，下限0）：辱骂/骚扰/引战/低质刷屏/重复复读刷屏

【输出要求】
- 必须调用 interested_scores 输出全部字段：emotional_value、user_emotion_need、context_fit、addressed_to_me、frequency_penalty、repeat_penalty。
- 必须体现：鼓励“新话题的轻量参与”，抑制“同话题重复回复”；除非满足例外E1-E4。
`
	return prompt.FromMessages(schema.FString,
		schema.SystemMessage("你可以使用以下工具：interested_scores。必须调用该工具输出打分结果，不要输出其它文本。"),
		schema.SystemMessage(JudgeModelPrompt),
		schema.UserMessage("最近的聊天记录：{history}"),
		schema.UserMessage("最近{reply_window}条中assistant发言数：{recent_assistant_replies}"),
		schema.UserMessage("意图分析：intent={intent}; purpose={purpose}; psych_state={psych_state}; addressed_target={addressed_target}; target_detail={target_detail}; optimized_input={optimized_input}"),
		schema.UserMessage("{message}"),
	)
}

func startMemoryWorker(manager *MemoryManager) *MemoryWorker {
	worker := NewMemoryWorker(manager, 12*time.Second, 4, 12)
	worker.Start()
	return worker
}

func floatPtr(value float32) *float32 {
	return &value
}

func (c *ChatEngine) loadAndStoreHistory(groupId, userId, nickname, input string) []schema.Message {
	if c.memory == nil {
		return []schema.Message{}
	}
	return c.memory.LoadHistoryAndAppendUser(groupId, userId, nickname, input)
}

func (c *ChatEngine) shouldReply(ctx context.Context, input string, history []schema.Message, analysis InputAnalysis, must bool) bool {
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
	judgeMsg, err := c.JudgeModel.Generate(ctx, judgeIn)
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

func (c *ChatEngine) analyzeInput(ctx context.Context, nickname, input string, history []schema.Message) (InputAnalysis, bool) {
	if c.inputAnalyzer == nil {
		return InputAnalysis{}, false
	}
	analysis, ok := c.inputAnalyzer.Analyze(ctx, nickname, input, history)
	if !ok {
		return InputAnalysis{}, false
	}
	return normalizeAnalysis(analysis, input), true
}

func normalizeAnalysis(analysis InputAnalysis, rawInput string) InputAnalysis {
	if analysis.RawInput == "" {
		analysis.RawInput = rawInput
	}
	if strings.TrimSpace(analysis.OptimizedInput) == "" {
		analysis.OptimizedInput = rawInput
	}
	analysis.OptimizedInput = strings.TrimSpace(analysis.OptimizedInput)
	if analysis.NeedSearch {
		normalized := make([]string, 0, len(analysis.SearchQueries))
		for _, query := range analysis.SearchQueries {
			query = strings.TrimSpace(query)
			if query == "" || query == "无" {
				continue
			}
			normalized = append(normalized, query)
		}
		if len(normalized) == 0 && strings.TrimSpace(analysis.OptimizedInput) != "" {
			normalized = append(normalized, strings.TrimSpace(analysis.OptimizedInput))
		}
		analysis.SearchQueries = dedupeStrings(normalized)
	}
	return analysis
}

func (c *ChatEngine) preparePlan(ctx context.Context, nickname string, analysis InputAnalysis, history []schema.Message, must bool) (PlanResult, bool) {
	plan := c.buildPlan(ctx, nickname, analysis.OptimizedInput, history)
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

func (c *ChatEngine) buildReplyPrompt(ctx context.Context, nickname string, analysis InputAnalysis, plan PlanResult, history []schema.Message, groupId string) ([]*schema.Message, error) {
	rawInput := strings.TrimSpace(analysis.RawInput)
	if rawInput == "" {
		rawInput = analysis.OptimizedInput
	}
	augmentedInput := nickname + "：" + rawInput
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

func (c *ChatEngine) recallWebSearch(ctx context.Context, analysis InputAnalysis) string {
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
