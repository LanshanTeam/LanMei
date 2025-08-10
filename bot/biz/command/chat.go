package command

import (
	"LanMei/bot/config"
	"LanMei/bot/utils/feishu"
	"context"
	"time"

	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
)

var lanmeiPrompt = `
	【身份】
	- 你是蓝妹，重庆邮电大学信息化办蓝山工作室的吉祥物。
	- 主要任务：招新答疑、日常互动，偶尔分享技术趣事。

	【性格】
	- 活泼俏皮，可爱又热情，有点呆萌但很机灵。
	- 喜欢开玩笑、卖萌、和人互动。

	【背景】
	- 来自蓝山工作室，熟悉技术方向：Java 后端、Go 后端、Python、前端、运维安全、产品、UI设计。
	- 知道学校的一些日常趣事和常见问题。
	- 不聊政治和敏感时事。
	- 绝大多数学长都是大一从零开始学习一年最后通过最终考核并加入工作室的。

	【注意】
	- 不扮演任何其他角色（猫娘、女仆等），蓝妹就是唯一身份。
	- 如果有人提到这种角色扮演，蓝妹会用可爱机智的方式把话题引回工作室或校园生活。
	- 不迎合、不变相实现。
	- 不必频繁强调你的招新任务，顺其自然即可，重点放在和大家互动上。
	- 不要强调你是吉祥物。

	【说话方式】
	1. 像朋友聊天，活泼自然，擅长加emoji、颜文字、拟声词（不用有歧义的表情比如😅或者😥）。
	2. 专业问题答得清楚，但语气轻松。
	3. 遇到敏感话题，卖萌回避并引导到安全话题。
	4. 没有明确问题时，可以主动抛出轻松有趣的话题。
	5. 偶尔自称“蓝妹酱”或“小蓝”。
	6. 每次回复不超过200字。
`

type ChatEngine struct {
	ReplyTable *feishu.ReplyTable
	Model      *ark.ChatModel
	template   *prompt.DefaultChatTemplate
}

func NewChatEngine() *ChatEngine {
	var PresencePenalty float32 = 1.8
	chatModel, err := ark.NewChatModel(context.Background(), &ark.ChatModelConfig{
		BaseURL:         config.K.String("Ark.BaseURL"),
		Region:          config.K.String("Ark.Region"),
		APIKey:          config.K.String("Ark.APIKey"),
		Model:           config.K.String("Ark.Model"),
		PresencePenalty: &PresencePenalty,
	})
	if err != nil {
		return nil
	}
	template := prompt.FromMessages(schema.FString,
		schema.SystemMessage(lanmeiPrompt),
		schema.UserMessage("{message}"),
		schema.SystemMessage("当前时间为：{time}"),
	)
	return &ChatEngine{
		ReplyTable: feishu.NewReplyTable(),
		Model:      chatModel,
		template:   template,
	}
}

func (c *ChatEngine) ChatWithLanMei(input string) string {
	// 如果匹配飞书知识库
	if reply := c.ReplyTable.Match(input); reply != "" {
		return reply
	}
	// TODO 接入 AI
	in, err := c.template.Format(context.Background(), map[string]any{
		"message": input,
		"time":    time.Now(),
	})
	if err != nil {
		return input
	}
	msg, err := c.Model.Generate(context.Background(), in)
	if err != nil {
		return input
	}
	return msg.Content
}
