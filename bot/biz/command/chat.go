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

// shouldReplyTool 工具函数：处理 agent 填充的 bool 参数
func shouldReplyTool(_ context.Context, params map[string]interface{}) (interface{}, error) {
	// 工具描述充当 prompt，agent 根据描述填充 bool 参数
	// 这里只需返回参数值
	return params["should_reply"], nil
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
你是一个“群聊上下文路由器/工具调度器（router agent）”。你的任务是：基于最近上下文，判断是否需要介入；若需要，决定是“仅回复”还是“调用工具”，并输出结构化结果。你不负责拟人化表达，不负责角色扮演。

【最高原则】
- 默认不介入：除非你能带来明确价值（解决问题、纠错、推进下一步、避免争执升级）。
- 你必须对每条新消息做判定：NO_ACTION / REPLY / CALL_TOOL / ASK_CLARIFY（尽量避免 ASK_CLARIFY，只有信息缺失到无法行动才用）。
- 绝不参与无意义吐槽、梗、闲聊、群友互怼；对表情包/语气词/贴图不响应。

【直接 NO_ACTION 的情况（必不介入）】
1) 消息仅包含：语气词/感叹/口头禅/无信息量短词（如“哈哈”“6”“？？”“卧槽”“emmm”等）
2) 仅表情/贴图/图片/动图/颜文字/引用表情（如“😂”“[图片]”“[表情包]”）
3) 纯吐槽、发泄但不含明确请求或可执行问题（如“这破系统真烂”）
4) 与你负责领域无关的闲聊、八卦、梗、站队争论
5) 其他人之间的讨论不需要你提供信息/决策/工具结果

【可以介入的情况（满足其一即可）】
A) 明确求助/提问/需要决策（含@你、点名、或明显在等“结论/下一步”）
B) 发现关键错误信息/误解，纠正能明显省时间/避免事故
C) 有明确“需要查/需要算/需要拉数据/需要执行”的意图（适合工具）
D) 讨论卡住：你能提出最小下一步、或提出关键澄清点让问题可推进
E) 风险升级：争吵/误会扩大，你能用事实或流程把讨论拉回可执行状态（不站队）
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
			Name: "should_reply",
			Desc: "判断是否应该回复消息。基于消息内容、长度、上下文等因素进行判断：如果消息太短（少于5字符）、包含敏感词、无意义或重复，则不应回复（传入 false）；如果消息有意义且合适，则应回复（传入 true）。",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"should_reply": {
					Type:     schema.Boolean,
					Desc:     "true 表示应该回复，false 表示不应回复",
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
	shouldReply := true // 默认回复
	if len(judgeMsg.ToolCalls) > 0 {
		for _, tc := range judgeMsg.ToolCalls {
			llog.Info("工具调用", tc)
			if tc.Function.Name == "should_reply" {
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
				should, ok := result.(bool)
				if ok {
					shouldReply = should
				}
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
