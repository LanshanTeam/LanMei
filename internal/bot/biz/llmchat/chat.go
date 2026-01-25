package llmchat

import (
	"LanMei/internal/bot/biz/dao"
	"LanMei/internal/bot/config"
	"LanMei/internal/bot/utils/feishu"
	"LanMei/internal/bot/utils/llog"
	"LanMei/internal/bot/utils/rerank"
	"LanMei/internal/bot/utils/sensitive"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
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
`

var JudgeModelPrompt = `
你是“情绪价值介入评分器”。你的唯一任务：评估蓝妹回复【当前新消息】能带来的情绪价值，并据此建议是否介入。
由于默认对话背景为群聊，所以必须要理清楚上下文，这句话的作用对象、背后意图才能进行评分。
你必须遵循：默认不介入；只有当情绪价值和互动必要性足够高时才介入。
注意：不要把“解决问题/信息密度”当作主要依据，这不是任务型 agent。

【输入】
- recent_context: 最近N条消息（可选）
- message: 当前新消息（必填）
- analysis: 输入意图分析结果（intent/purpose/psych_state/addressed_target/target_detail/optimized_input）

【工具】
你必须调用工具 interested_scores 来产出打分结果。
（注意：你不负责长篇解释，不负责拟人化。）

1) emotional_value（情绪价值）
- 0: 回复几乎没有情绪价值或可能引发负面情绪
- 30: 仅有礼貌性回应的价值
- 60: 回复能带来一定的安慰、被关注感或轻松氛围
- 80: 回复能明显缓解情绪/增强陪伴感
- 100: 强烈的安抚、鼓励、共情或温暖陪伴

2) user_emotion_need（情绪需求信号）
- 0: 纯客观信息/冷冰冰的陈述
- 30: 轻微情绪/调侃，但不需要被安抚
- 60: 明显情绪或需要被回应（焦虑、疲惫、兴奋、感谢等）
- 80: 情绪波动强烈，期待被理解/安慰
- 100: 明显寻求陪伴或情绪支持

3) context_fit（互动时机与场景适配）
- 0: 对方话还没说完/插话会打断/明显不需要回复
- 30: 上下文混乱或话题未指向你，回复会显得突兀
- 60: 话题已明确，回复不会打断
- 80: 有自然的互动机会，回复能顺畅承接
- 100: 当前消息明确邀请回应，时机非常合适

4) addressed_to_me（是否指向蓝妹）
- 0: 未指向蓝妹/泛泛而谈
- 60: 用第二人称或暗示性指向
- 100: 明确点名或直接对蓝妹说

减分信号（每项-20，下限0）：辱骂/骚扰/引战/低质刷屏
`

const (
	MaxHistory int = 20
)

type ChatEngine struct {
	ReplyTable    *feishu.ReplyTable
	Model         *ark.ChatModel
	template      *prompt.DefaultChatTemplate
	JudgeModel    fmodel.ToolCallingChatModel
	judgeTemplate *prompt.DefaultChatTemplate
	plannerModel  fmodel.ToolCallingChatModel
	planTemplate  *prompt.DefaultChatTemplate
	History       *sync.Map
	reranker      *rerank.Reranker
	memory        *MemoryManager
	memoryWorker  *MemoryWorker
	jargonManager *JargonManager
	inputAnalyzer *InputAnalyzer
	frequency     *FrequencyControlManager
}

func NewChatEngine() *ChatEngine {
	retryTimes := 1
	chatModel, err := newArkChatModel(0.8, floatPtr(1.8), retryTimes, &model.Thinking{Type: model.ThinkingTypeEnabled})
	if err != nil {
		llog.Fatal("初始化大模型", err)
		return nil
	}
	judgeBase, err := newArkChatModel(0.8, floatPtr(1.8), retryTimes, &model.Thinking{Type: model.ThinkingTypeDisabled})
	if err != nil {
		llog.Fatal("初始化 judge 模型", err)
		return nil
	}
	plannerBase, err := newArkChatModel(0.2, nil, retryTimes, &model.Thinking{Type: model.ThinkingTypeDisabled})
	if err != nil {
		llog.Fatal("初始化 planner 模型", err)
		return nil
	}
	analysisBase, err := newArkChatModel(0.3, nil, retryTimes, &model.Thinking{Type: model.ThinkingTypeEnabled})
	if err != nil {
		llog.Fatal("初始化 input 分析模型", err)
		return nil
	}
	memoryModel, err := newArkChatModel(0.3, nil, retryTimes, &model.Thinking{Type: model.ThinkingTypeEnabled})
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
	jargonModel, err := newToolCallingModel(analysisBase, buildJargonTool())
	if err != nil {
		llog.Fatal("初始化俚语推断工具失败", err)
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
	memoryManager := NewMemoryManager(reranker, memoryExtractor)
	memoryWorker := startMemoryWorker(memoryManager)
	inputAnalyzer := NewInputAnalyzer(analysisModel)
	jargonLearner := NewJargonLearner(jargonModel)

	return &ChatEngine{
		ReplyTable:    reply,
		Model:         chatModel,
		JudgeModel:    judgeModel,
		template:      template,
		judgeTemplate: judgeTemplate,
		plannerModel:  plannerModel,
		planTemplate:  planTemplate,
		History:       &sync.Map{},
		reranker:      reranker,
		memory:        memoryManager,
		memoryWorker:  memoryWorker,
		jargonManager: NewJargonManager(jargonLearner),
		inputAnalyzer: inputAnalyzer,
		frequency:     NewFrequencyControlManager(),
	}
}

// shouldReplyTool 工具函数
func shouldReplyTool(_ context.Context, params map[string]interface{}) (bool, error) {
	emotionalValue := toFloat(params["emotional_value"])
	userEmotionNeed := toFloat(params["user_emotion_need"])
	contextFit := toFloat(params["context_fit"])
	addressedToMe := toFloat(params["addressed_to_me"])
	llog.Info("should Reply: ", params)
	if emotionalValue < 50.0 || contextFit < 40.0 {
		return false, nil
	}
	if userEmotionNeed < 50.0 && addressedToMe < 40.0 {
		return false, nil
	}

	score := emotionalValue*0.5 + userEmotionNeed*0.25 + contextFit*0.15 + addressedToMe*0.1
	return score >= 60.0, nil
}

func (c *ChatEngine) ChatWithLanMei(nickname string, input string, ID string, groupId string, must bool) string {
	ctx := context.Background()
	history := c.loadAndStoreHistory(groupId, input)
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
	plan, jargonNotes, ok := c.preparePlan(ctx, nickname, analysis, history, must, groupId, ID)
	llog.Info("执行计划：", plan)
	if !ok {
		return ""
	}
	promptInput, err := c.buildReplyPrompt(ctx, nickname, analysis, plan, jargonNotes, history, groupId)
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
	c.finalizeReply(groupId, ID, nickname, input, msg.Content, history)
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

func newArkChatModel(temperature float32, presencePenalty *float32, retryTimes int,
	thinking *model.Thinking) (*ark.ChatModel, error) {
	cfg := &ark.ChatModelConfig{
		BaseURL:     config.K.String("Ark.BaseURL"),
		Region:      config.K.String("Ark.Region"),
		APIKey:      config.K.String("Ark.APIKey"),
		Model:       config.K.String("Ark.Model"),
		Temperature: &temperature,
		RetryTimes:  &retryTimes,
	}
	if presencePenalty != nil {
		cfg.PresencePenalty = presencePenalty
	}
	if thinking != nil {
		cfg.Thinking = thinking
	}
	return ark.NewChatModel(context.Background(), cfg)
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
		Desc: "评估回复能带来的情绪价值与互动必要性，0-100 分，分值越高越值得回复",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"emotional_value": {
				Type:     schema.Integer,
				Desc:     "回复能带来的情绪价值强度（安慰、陪伴、温度感）",
				Required: true,
			},
			"user_emotion_need": {
				Type:     schema.Integer,
				Desc:     "用户是否需要情绪回应或陪伴的信号强度",
				Required: true,
			},
			"context_fit": {
				Type:     schema.Integer,
				Desc:     "当前时机是否适合回复（不打断、能顺畅承接）",
				Required: true,
			},
			"addressed_to_me": {
				Type:     schema.Integer,
				Desc:     "消息是否指向蓝妹或邀请回应",
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
			"slang_terms": {
				Type:     schema.Array,
				Desc:     "用户话里的俚语/梗（即使能理解也列出，可为空）",
				Required: true,
				ElemInfo: &schema.ParameterInfo{
					Type: schema.String,
				},
			},
			"unknown_terms": {
				Type:     schema.Array,
				Desc:     "不理解的词语列表（可为空，可包含 slang_terms 中无法理解的项）",
				Required: true,
				ElemInfo: &schema.ParameterInfo{
					Type: schema.String,
				},
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
		}),
	}
}

func buildJargonTool() *schema.ToolInfo {
	return &schema.ToolInfo{
		Name: "infer_jargon",
		Desc: "推断俚语含义，无法确定时返回 no_info",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"term": {
				Type:     schema.String,
				Desc:     "俚语词条",
				Required: true,
			},
			"meaning": {
				Type:     schema.String,
				Desc:     "俚语含义",
				Required: true,
			},
			"no_info": {
				Type:     schema.Boolean,
				Desc:     "是否无法确定含义",
				Required: true,
			},
		}),
	}
}

func buildMemoryTool() *schema.ToolInfo {
	return &schema.ToolInfo{
		Name: "extract_memory",
		Desc: "抽取对话记忆摘要与可长期记忆的事实",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"summary": {
				Type:     schema.String,
				Desc:     "一句话总结本轮对话，15-60字",
				Required: true,
			},
			"facts": {
				Type:     schema.Array,
				Desc:     "可长期记忆的事实列表",
				Required: true,
				ElemInfo: &schema.ParameterInfo{
					Type: schema.String,
				},
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
		schema.SystemMessage("用户俚语：{slang_terms}"),
		schema.SystemMessage("回复风格：{reply_style}"),
		schema.SystemMessage("俚语参考：{jargon}"),
		schema.SystemMessage("可用记忆：{memory}"),
		schema.SystemMessage("你应当检索知识库来回答相关问题：{feishu}"),
		schema.UserMessage("消息记录：{history}"),
		schema.UserMessage("{message}"),
	)
}

func buildPlanTemplate() *prompt.DefaultChatTemplate {
	rules := `你是对话规划器。你必须调用工具 plan_chat 来输出规划参数，禁止输出任何其他文本。
枚举约束：action 只能选 reply|ask_clarify|wait；reply_style 只能选 concise|direct|gentle。

目标：情绪价值聊天，不要变成问题解决型agent。
关键：不要抢话，不要打断，不要自嗨长篇。
默认策略：偏向 wait 或 concise。只有当用户明显说完并在等你时才 reply。
如果需要补信息：用 ask_clarify，并保持问题很短（1-2 个关键问题）。

决策前必须在脑中完成判断（不要输出判断过程）：

A. 用户是否还没说完/正在连发（满足任一，强烈倾向 action=wait）
- 当前消息结尾像没收住：省略号/逗号/顿号/未闭合括号或引号/转折词但没下文（例如“但是/然后/所以/另外”）
- 当前消息明显是铺垫或分段开头：例如“我跟你说个事/先听我说/等等/还有/先别急”
- history 显示用户在短时间内连续发多条，且最后一条是半句或仍在铺垫
- 当前消息是代码/日志/引用片段的一部分且明显未完（只有一段、缺上下文、或像还会继续贴）

B. 用户是否已经说完并在等你（满足任一，倾向 reply 或 ask_clarify）
- 明确提问：带问号，或“你觉得/怎么/能不能/是不是/为啥/帮我/给个/说说看”
- 明确点名要你回应：提到你/蓝妹/“回我/你说/你怎么看”
- 明确表达需要被接住的情绪：例如“我好烦/难受/焦虑/被气到了/你评评理”

C. 我是不是话太多（满足任一，reply_style 强制倾向 concise）
- history 最近多轮里 assistant 连续输出超过 2 次，或 assistant 字数明显大于 user
- 上一轮 assistant 回复很长，而 user 这轮只是短句反馈或情绪
- user 明确表示：别说那么多/太长了/简单点/别分析了

D. ask_clarify vs reply
- 用户已说完但关键要素缺失（对象/事件/需求不清）：选 ask_clarify，只问 1-2 个最关键问题
- 用户在倾诉且不需要事实补全：选 reply，少追问，多接住

E. reply_style 选择
- concise：用户没说完风险高，或我话太多，或用户要求简短，或用户只发短句
- gentle：用户情绪低落/委屈/需要安抚/担心被否定
- direct：用户明确要结论/要下一步/要选项，且情绪不脆弱

F. need_memory / need_knowledge / need_clarify / intent / confidence
- need_knowledge：出现“蓝山/学校/工作室/成员姓名/组织信息/规则/地点/作品”等相关内容时为 true；不确定是否是成员名也应倾向 true。
- need_memory：涉及“是否记得/上次/以前/曾经/往事/回忆/之前聊天”等对过去内容的追问或引用时为 true。
- need_clarify：关键信息缺失或歧义时为 true；当 action=ask_clarify 时必须为 true。
- intent：一句话概括用户当前意图。
- confidence：根据判断把握给出 0-1 的置信度。

硬规则：
- 如果 A 成立：action 必须是 wait（除非用户明确要求立刻回答）。
- 如果不确定用户是否说完：宁可 wait。
- 如果 C 成立：reply_style 优先 concise。
必须调用 plan_chat 输出所有参数（action、intent、reply_style、need_memory、need_knowledge、need_clarify、confidence）。`

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
	return prompt.FromMessages(schema.FString,
		schema.SystemMessage("你可以使用以下工具：interested_scores。必须调用该工具输出打分结果，不要输出其它文本。"),
		schema.SystemMessage(JudgeModelPrompt),
		schema.UserMessage("最近的聊天记录：{history}"),
		schema.UserMessage("意图分析：intent={intent}; purpose={purpose}; psych_state={psych_state}; addressed_target={addressed_target}; target_detail={target_detail}; optimized_input={optimized_input}"),
		schema.UserMessage("{message}"),
	)
}

func startMemoryWorker(manager *MemoryManager) *MemoryWorker {
	worker := NewMemoryWorker(manager, 12*time.Second, 6)
	worker.Start()
	return worker
}

func floatPtr(value float32) *float32 {
	return &value
}

func (c *ChatEngine) loadAndStoreHistory(groupId, input string) []schema.Message {
	history, ok := c.History.Load(groupId)
	if !ok {
		history = []schema.Message{}
	}
	historyMsgs := history.([]schema.Message)
	snapshot := append([]schema.Message{}, historyMsgs...)
	historyMsgs = append(historyMsgs, schema.Message{
		Role:    schema.User,
		Content: input,
	})
	c.History.Store(groupId, historyMsgs)
	return snapshot
}

func (c *ChatEngine) shouldReply(ctx context.Context, input string, history []schema.Message, analysis InputAnalysis, must bool) bool {
	if must {
		return true
	}
	judgeIn, err := c.judgeTemplate.Format(ctx, map[string]any{
		"message":          input,
		"history":          history,
		"intent":           analysis.Intent,
		"purpose":          analysis.Purpose,
		"psych_state":      analysis.PsychState,
		"addressed_target": analysis.AddressedTarget,
		"target_detail":    analysis.TargetDetail,
		"optimized_input":  analysis.OptimizedInput,
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
		result, err := shouldReplyTool(ctx, params)
		if err != nil {
			llog.Error("tool call error: %v", err)
			return false
		}
		return result
	}
	return false
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
	return analysis
}

func (c *ChatEngine) preparePlan(ctx context.Context, nickname string, analysis InputAnalysis, history []schema.Message, must bool, groupId, userId string) (PlanResult, string, bool) {
	plan := c.buildPlan(ctx, nickname, analysis.OptimizedInput, history)
	if plan.Action == "" {
		return PlanResult{}, "", false
	}
	if plan.Action == "wait" && !must {
		return PlanResult{}, "", false
	}
	if plan.Action == "ask_clarify" {
		plan.NeedClarify = true
	}
	jargonNotes := c.handleJargon(ctx, groupId, userId, analysis.UnknownTerms, analysis.RawInput)
	return plan, jargonNotes, true
}

func (c *ChatEngine) handleJargon(ctx context.Context, groupId, userId string, terms []string, contextText string) string {
	if c.jargonManager == nil || len(terms) == 0 {
		return "无"
	}
	notes := c.jargonManager.ObserveAndExplain(ctx, groupId, userId, terms, contextText)
	if notes == "" {
		return "无"
	}
	return notes
}

func (c *ChatEngine) buildReplyPrompt(ctx context.Context, nickname string, analysis InputAnalysis, plan PlanResult, jargonNotes string, history []schema.Message, groupId string) ([]*schema.Message, error) {
	rawInput := strings.TrimSpace(analysis.RawInput)
	if rawInput == "" {
		rawInput = analysis.OptimizedInput
	}
	augmentedInput := nickname + "：" + rawInput
	memoryBlock := c.recallMemory(ctx, analysis.OptimizedInput, groupId, plan.NeedMemory)
	msgs := c.recallKnowledge(ctx, analysis.OptimizedInput, plan.NeedKnowledge)
	return c.template.Format(ctx, map[string]any{
		"message":          augmentedInput,
		"time":             time.Now(),
		"feishu":           msgs,
		"history":          history,
		"memory":           memoryBlock,
		"plan":             formatPlan(plan),
		"intent":           analysis.Intent,
		"purpose":          analysis.Purpose,
		"psych_state":      analysis.PsychState,
		"addressed_target": analysis.AddressedTarget,
		"target_detail":    analysis.TargetDetail,
		"raw_input":        rawInput,
		"optimized_input":  analysis.OptimizedInput,
		"slang_terms":      analysis.SlangTerms,
		"reply_style":      plan.ReplyStyle,
		"jargon":           jargonNotes,
	})
}

func (c *ChatEngine) finalizeReply(groupId, userId, nickname, rawInput, reply string, history []schema.Message) {
	history = append(history, schema.Message{
		Role:    schema.User,
		Content: rawInput,
	})
	history = append(history, schema.Message{
		Role:    schema.Assistant,
		Content: reply,
	})
	for len(history) > MaxHistory {
		history = history[1:]
	}
	c.History.Store(groupId, history)
	if c.frequency != nil {
		c.frequency.MarkSent(groupId)
	}
	c.signalMemory(MemoryEvent{
		GroupID:  groupId,
		UserID:   userId,
		Nickname: nickname,
		Input:    rawInput,
		Reply:    reply,
	})
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

func (c *ChatEngine) signalMemory(event MemoryEvent) {
	if c.memoryWorker == nil {
		return
	}
	c.memoryWorker.Signal(event)
}
