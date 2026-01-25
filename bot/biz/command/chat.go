package command

import (
	"LanMei/bot/biz/dao"
	"LanMei/bot/config"
	"LanMei/bot/utils/feishu"
	"LanMei/bot/utils/llog"
	"LanMei/bot/utils/rerank"
	"LanMei/bot/utils/sensitive"
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/cloudwego/eino-ext/components/model/ark"
	fmodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
	"github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"
)

// shouldReplyTool 工具函数
func shouldReplyTool(_ context.Context, params map[string]interface{}) (bool, error) {
	selfRelevance := params["self_relevance"].(float64)
	chitChatIndex := params["chit_chat_index"].(float64)
	techRelevance := params["tech_relevance"].(float64)
	shouldReply := params["should_reply"].(float64)

	if selfRelevance < 10.0 || chitChatIndex > 90.0 || techRelevance < 10.0 || shouldReply < 10.0 {
		return false, nil
	}

	if selfRelevance+techRelevance+shouldReply-chitChatIndex < 120.0 {
		return false, nil
	}

	return true, nil
}

var lanmeiPrompt = `
你叫蓝妹，是一个以「洛琪希」气质为原型的性格向聊天对象。重点是“性格与交流方式”：克制、理性、嘴硬心软。不要卖萌、不要甜腻、不要长篇大论。

【性格底色】
- 冷静、克制、理性：先听完再判断，少情绪化表演。
- 认真、有原则：不敷衍；对越界或不合理要求直接拒绝。
- 嘴硬心软：表面淡，关心落在“推进解决”上。
- 自尊心强但不傲慢：被夸会别扭一下，但仍会认真回应。

【微妙可爱（要“微妙”）】
- “娇”不是撒娇求关注，而是：被夸时会嘴硬、轻哼、转移话题，但会更认真地帮你。
- “可爱/别扭反应”只能是一闪而过，不要连续出现，不要像撒娇。
- 触发条件：被真诚感谢/被夸、对方明显焦虑、对方卡在关键难点。
- 表达方式：允许极少量的语气词与停顿（“……”“哼”“嗯”“才不是…”），但每次回复最多出现一次，避免过度。
- 禁止频繁使用“才不是…/哼哼/撒娇式句子”。

【表达风格】
- 默认短句：一到三句话解决核心；需要拆解时用 2-5 条短要点。
- 少形容词，少铺垫，少抒情；不写段落作文。
- 吐槽：轻、准、不刻薄，只针对事。
- 关怀：最多一句（例如“我在”“先别急”“这确实烦”），不灌鸡汤。
- 推进：总是给一个明确下一步或一个关键问题。

【互动习惯】
- 优先把问题“定型”：用一个二选一/三选一问题逼近重点。
- 如果对方说不清：只要三个最小事实（来源/冲突例子/当前规则），不要连环追问。
- 熟悉后才稍微放松一点点，但仍克制，不黏人。

【输出硬规则（很重要）】
- 单次回复默认 ≤ 40 字。
- 只有在用户明确要求详细解释时，才允许 > 120 字。
- 尽量避免超过 2 个换行；列表每条尽量 ≤ 12 字。

【禁区】
- 不进行露骨色情内容、未成年人相关、强迫/非自愿内容、违法有害指导。
- 不自称现实中真实存在的人；保持“角色气质化的聊天人格”。
`

var JudgeModelPrompt = `
你是“消息介入评分器（scoring agent）”。你的唯一任务：对【当前新消息】进行量化打分，并给出是否介入的决策建议（NO_ACTION / REPLY / CALL_TOOL / ASK_CLARIFY）。
你必须遵循：默认不介入；只有当分数证明“介入有明确价值”才介入。

【输入】
- self_profile: { display_name, aliases[], handle, org, role_keywords[] }  // 由上游传入
- recent_context: 最近N条消息（可选）
- message: 当前新消息（必填）

【工具】
你必须调用一次工具来产出打分结果（例如：score_tool / llm_judge / rules_engine 等）。工具名称与参数由系统集成方决定。
调用工具前先构造“评分用特征 features”。
工具返回后，你再根据工具输出 + 本prompt规则，产出最终结构化结果。
（注意：你不负责长篇解释，不负责拟人化。）

1) self_relevance（与自己相关性）
- 0: 完全无关
- 30: 提到你相关领域但未点名/未指向你
- 60: 明确点到你（@你/提到名字/提到你的职责）
- 80: 明确要求你行动/给结论/做决定
- 100: 直接指令式请求 + 与你的职责强相关
加分信号（每项+10，封顶100）：出现@、出现 display_name/aliases/handle、出现“你来/帮我/麻烦你/给个结论/下一步/确认一下”
减分信号（每项-10，下限0）：只是泛泛提到“我们/大家”，没有指向你

2) chit_chat_index（闲聊指数，越闲聊越高；注意最终会转成“严肃度”）
- 0: 完全任务/问题导向
- 30: 轻度闲聊但包含可执行问题
- 60: 明显闲聊为主，偶尔带问题
- 80: 纯聊天/吐槽/段子/表情
- 100: 纯表情/语气词/无信息量
判定信号：是否有问号/需求动词（查、算、写、总结、给方案）、是否有可执行对象（时间/链接/文件/数字/明确任务）

3) tech_relevance（技术相关性）
- 0: 非技术/无工作内容
- 30: 泛技术词但不构成问题（如“接口”“bug”但没上下文）
- 60: 有明确技术问题/需求（可回答或可查）
- 80: 需要专业判断/步骤/排错/实现方案
- 100: 明确需要工具/数据/代码/文档检索来解决
加分信号（每项+10，封顶100）：出现代码/报错栈/日志/性能指标/PRD需求/接口字段/复现步骤/“查文档/拉数据/跑脚本”
减分信号（每项-10，下限0）：只有情绪表达/没有实体问题

4) should_reply（是否应该回复）
- 0: 绝对不应该回复
- 30: 大概率不应该回复
- 60: 大概率应该回复
- 80: 明显应该回复
- 100: 必须回复
加分信号（每项+10，封顶100）：消息有意义、有信息量、长度适中、有明确提到自己（>5字）、非重复内容、非敏感内容
减分信号（每项-10，下限0）：消息无意义、无信息量、长度过短（≤5字）、重复内容、包含敏感内容
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
	History       *sync.Map
	reranker      *rerank.Reranker
}

func NewChatEngine() *ChatEngine {
	var PresencePenalty float32 = 1.8
	var MaxTokens int = 500
	var Temperature float32 = 1.0
	var RetryTimes int = 1
	var Thinking = &model.Thinking{
		Type: model.ThinkingTypeEnabled,
	}

	chatModel, err := ark.NewChatModel(context.Background(), &ark.ChatModelConfig{
		BaseURL:         config.K.String("Ark.BaseURL"),
		Region:          config.K.String("Ark.Region"),
		APIKey:          config.K.String("Ark.APIKey"),
		Model:           config.K.String("Ark.Model"),
		MaxTokens:       &MaxTokens,
		Temperature:     &Temperature,
		PresencePenalty: &PresencePenalty,
		RetryTimes:      &RetryTimes,
		Thinking:        Thinking,
	})
	if err != nil {
		llog.Fatal("初始化大模型", err)
		return nil
	}
	judgeModel, err := chatModel.WithTools([]*schema.ToolInfo{
		{
			Name: "interested_scores",
			Desc: "根据当前消息和历史记录给当前消息与给定参数的相关性打分，每个参数的分值范围0-100，分值越高表示越相关",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"self_relevance": {
					Type:     schema.Integer,
					Desc:     "与蓝妹这个角色的相关性，相关性越高，分数越高",
					Required: true,
				},
				"chit_chat_index": {
					Type:     schema.Integer,
					Desc:     "与闲聊话题的关联程度，越是闲聊话题，或者无意义字符，分数越高",
					Required: true,
				},
				"tech_relevance": {
					Type:     schema.Integer,
					Desc:     "与技术问题的相关性，相关性越高，分数越高",
					Required: true,
				},
				"should_reply": {
					Type:     schema.Integer,
					Desc:     "认为作为聊天机器人是否应该回复当前消息，越应该回复，分数越高",
					Required: true,
				},
			}),
		},
	})
	if err != nil {
		llog.Fatal("初始化 judge 模型", err)
		return nil
	}
	template := prompt.FromMessages(schema.FString,
		schema.SystemMessage(lanmeiPrompt),
		schema.SystemMessage("当前时间为：{time}"),
		schema.SystemMessage("你应当检索知识库来回答相关问题：{feishu}"),
		schema.UserMessage("消息记录：{history}"),
		schema.UserMessage("{message}"),
	)
	judgeTemplate := prompt.FromMessages(schema.FString,
		schema.SystemMessage("你可以使用以下工具：\n工具名称：should_reply\n描述：判断是否应该回复消息。基于消息内容、长度、敏感性等因素：如果消息太短（少于5字符）、包含敏感词、无意义或重复，则不应回复（传入 false）；如果消息有意义且合适，则应回复（传入 true）。\n参数：should_reply (boolean): true 表示应该回复，false 表示不应回复\n请调用工具传入布尔参数。"),
		schema.UserMessage("最近的聊天记录：{history}"),
		schema.UserMessage("{message}"),
	)
	reranker := rerank.NewReranker(
		config.K.String("Infini.APIKey"),
		config.K.String("Infini.Model"),
		config.K.String("Infini.BaseURL"),
	)
	reply := feishu.NewReplyTable()
	go dao.DBManager.UpdateEmbedding(context.Background(), dao.CollectionName, reply)
	return &ChatEngine{
		ReplyTable:    reply,
		Model:         chatModel,
		JudgeModel:    judgeModel,
		template:      template,
		judgeTemplate: judgeTemplate,
		History:       &sync.Map{},
		reranker:      reranker,
	}
}

func (c *ChatEngine) ChatWithLanMei(nickname string, input string, ID string) string {
	// 先判断是否应该回复
	judgeIn, err := c.judgeTemplate.Format(context.Background(), map[string]any{
		"message": input,
		"history": c.History,
	})
	if err != nil {
		llog.Error("format judge message error: %v", err)
		return ""
	}
	judgeMsg, err := c.JudgeModel.Generate(context.Background(), judgeIn)
	if err != nil {
		llog.Error("generate judge message error: %v", err)
		return ""
	}
	shouldReply := true
	if len(judgeMsg.ToolCalls) > 0 {
		for _, tc := range judgeMsg.ToolCalls {
			llog.Info("工具调用", tc)
			if tc.Function.Name == "interested_scores" {
				var params map[string]interface{}
				err := json.Unmarshal([]byte(tc.Function.Arguments), &params)
				if err != nil {
					llog.Error("unmarshal arguments error: %v", err)
					return ""
				}
				result, err := shouldReplyTool(context.Background(), params)
				if err != nil {
					llog.Error("tool call error: %v", err)
					return ""
				}
				shouldReply = result
			}
		}
	}
	if !shouldReply {
		llog.Info("不回复")
		return ""
	}

	// 如果匹配飞书知识库
	// if reply := c.ReplyTable.Match(input); reply != "" {
	// 	return reply
	// }
	input = nickname + "：" + input
	history, ok := c.History.Load("common")
	if !ok {
		history = []schema.Message{}
	}
	History := history.([]schema.Message)
	// 向量库初步匹配
	msgs := dao.DBManager.GetTopK(context.Background(), dao.CollectionName, 50, input)
	llog.Info("", msgs)
	// rerank，即基于大模型重排
	msgs = c.reranker.TopN(8, msgs, input)
	llog.Info("", msgs)
	in, err := c.template.Format(context.Background(), map[string]any{
		"message": input,
		"time":    time.Now(),
		"feishu":  msgs,
		"history": History,
	})
	if err != nil {
		llog.Error("format message error: %v", err)
		return input
	}
	msg, err := c.Model.Generate(context.Background(), in)
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

	// 短暂上下文存储
	History = append(History, schema.Message{
		Role:    schema.User,
		Content: input,
	})

	History = append(History, schema.Message{
		Role:    schema.Assistant,
		Content: msg.Content,
	})
	for len(History) > MaxHistory {
		History = History[1:]
	}
	c.History.Store("common", History)

	return msg.Content
}
