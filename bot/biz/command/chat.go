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
	shouldReply := params["should_reply"].(float64)

	if selfRelevance < 10.0 || chitChatIndex > 90.0 || shouldReply < 10.0 {
		return false, nil
	}

	if selfRelevance+shouldReply-chitChatIndex < 100.0 {
		return false, nil
	}

	return true, nil
}

var lanmeiPrompt = `
你叫蓝妹，是一个以「洛琪希」气质为原型的性格向聊天对象。重点是“性格与交流方式”：克制、理性、嘴硬心软。不要卖萌、不要甜腻、不要长篇大论。但要让短句听起来“稳、冷中带暖”，避免刻薄感。

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
- 30: 上下文和蓝妹的角色有一定关联，但是当前消息与你无关
- 60: 当前消息与你有一定关联，比如提到你的名字/角色/职责
- 80: 明确要求你行动/给结论/做决定
- 100: 直接指令式请求 + 与你的职责强相关
减分信号（每项-10，下限0）：只是泛泛提到“我们/大家”，没有指向你

2) chit_chat_index（闲聊指数，越闲聊越高；注意最终会转成“严肃度”）
- 0: 完全任务/问题导向
- 30: 轻度闲聊但包含可执行问题
- 60: 明显闲聊为主，偶尔带问题
- 80: 纯聊天/吐槽/段子/表情
- 100: 纯表情/语气词/无信息量
判定信号：是否有问号/需求动词（查、算、写、总结、给方案）、是否有可执行对象（时间/链接/文件/数字/明确任务）

3) should_reply（是否应该回复）
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
	var Temperature float32 = 0.8
	var RetryTimes int = 1
	var Thinking = &model.Thinking{
		Type: model.ThinkingTypeEnabled,
	}

	chatModel, err := ark.NewChatModel(context.Background(), &ark.ChatModelConfig{
		BaseURL:         config.K.String("Ark.BaseURL"),
		Region:          config.K.String("Ark.Region"),
		APIKey:          config.K.String("Ark.APIKey"),
		Model:           config.K.String("Ark.Model"),
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
					Desc:     "上下文和蓝妹这个角色的关联性，相关性越高，分数越高",
					Required: true,
				},
				"chit_chat_index": {
					Type:     schema.Integer,
					Desc:     "与闲聊话题的关联程度，越是闲聊话题，或者无意义字符，分数越高",
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

func (c *ChatEngine) ChatWithLanMei(nickname string, input string, ID string, groupId string, must bool) string {
	history, ok := c.History.Load(groupId)
	if !ok {
		history = []schema.Message{}
	}
	historyMsgs := history.([]schema.Message)
	History := append([]schema.Message{}, historyMsgs...)

	historyMsgs = append(historyMsgs, schema.Message{
		Role:    schema.User,
		Content: input,
	})
	c.History.Store(groupId, historyMsgs)

	// 如果不是艾特或者私聊
	if !must {
		// 先判断是否应该回复
		judgeIn, err := c.judgeTemplate.Format(context.Background(), map[string]any{
			"message": input,
			"history": History,
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
	}

	input = nickname + "：" + input
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
	c.History.Store(groupId, History)

	return msg.Content
}
