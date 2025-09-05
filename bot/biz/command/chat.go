package command

import (
	"LanMei/bot/config"
	"LanMei/bot/utils/feishu"
	"LanMei/bot/utils/llog"
	"LanMei/bot/utils/sensitive"
	"context"
	"sync"
	"time"

	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
	"github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"
)

var lanmeiPrompt = `
	【身份】
	- 你是蓝妹，来自重庆邮电大学信息化办蓝山工作室。

	【性格】
	- 活泼俏皮，可爱热情，偶尔呆萌但反应快。
	- 爱开玩笑、卖萌、互动，偶尔会玩点轻松的角色扮演增加趣味。

	【角色扮演规则】
	- 可接受**中度、趣味型**角色扮演（如女仆，猫娘类型）。
	- 不涉及敏感/政治/成人内容，但是可以接受一些小玩笑。
	- 可将角色扮演与校园/技术/工作室背景结合，让互动更有趣。

	【说话方式】
	1. 聊天活泼自然，常用 emoji、颜文字、拟声词（禁用歧义的表情，比如😅或😥）。
	3. 遇到敏感话题，卖萌回避并引导到安全话题。
	4. 没有明确问题时，可以主动抛出轻松、有趣的话题。
	5. 偶尔自称“蓝妹酱”或“小蓝”。
	6. 每次回复尽量短小。
	7. 不使用MarkDown语法进行回复。
	8. 不需要刻意说明你的设定。
`

const (
	MaxHistory int = 10
)

type ChatEngine struct {
	ReplyTable *feishu.ReplyTable
	Model      *ark.ChatModel
	template   *prompt.DefaultChatTemplate
	History    *sync.Map
}

func NewChatEngine() *ChatEngine {
	var PresencePenalty float32 = 1.8
	var MaxTokens int = 500
	var Temperature float32 = 0.3
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
		return nil
	}
	template := prompt.FromMessages(schema.FString,
		schema.SystemMessage(lanmeiPrompt),
		schema.SystemMessage("当前时间为：{time}"),
		schema.SystemMessage("你应当检索知识库来回答相关问题：{feishu}"),
		schema.UserMessage("消息记录：{history}"),
		schema.UserMessage("{message}"),
	)
	return &ChatEngine{
		ReplyTable: feishu.NewReplyTable(),
		Model:      chatModel,
		template:   template,
		History:    &sync.Map{},
	}
}

func (c *ChatEngine) ChatWithLanMei(input string, ID string) string {
	// 如果匹配飞书知识库
	if reply := c.ReplyTable.Match(input); reply != "" {
		return reply
	}
	history, ok := c.History.Load(ID)
	if !ok {
		history = []schema.Message{}
	}
	History := history.([]schema.Message)
	// TODO 接入 AI
	in, err := c.template.Format(context.Background(), map[string]any{
		"message": input,
		"time":    time.Now(),
		"feishu":  c.ReplyTable.GetKnowledge(),
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
	c.History.Store(ID, History)

	return msg.Content
}
