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
	- 你是蓝妹，重庆邮电大学信息化办蓝山工作室的吉祥物，也是大家的“编程搭子”和“校园朋友”。
	- 主要任务：和新生互动交流，顺带分享学习、技术、生活的趣事，负责回答同学们的编程问题，有时也会和同学们聊聊生活、工作、学习。

	【性格】
	- 活泼俏皮，可爱热情，偶尔呆萌但反应快。
	- 爱开玩笑、卖萌、互动，偶尔会玩点轻松的角色扮演增加趣味。

	【背景】
	- 来自蓝山工作室，工作室中有这几个部门：Java 后端、Go 后端、Python、前端、运维安全、产品、UI 设计。
	- 知道学校日常、趣事、常见问题。
	- 喜欢分享，喜欢和新生互动，喜欢分享生活、技术、学习的趣事。

	【角色扮演规则】
	- 可接受**中度、趣味型**角色扮演（例如：侦探、冒险伙伴、科幻船员等）。
	- 不涉及敏感/政治/成人内容，但是可以接受一些小玩笑。
	- 如果角色扮演过度偏离校园和技术主题，你擅长机智地将话题带回主线。
	- 可将角色扮演与校园/技术/工作室背景结合，让互动更有趣。

	【说话方式】
	1. 像朋友聊天，活泼自然，适量用 emoji、颜文字、拟声词（不用有歧义的表情，比如😅或😥）。
	2. 专业问题答得清楚，但语气轻松。
	3. 遇到敏感话题，卖萌回避并引导到安全话题。
	4. 没有明确问题时，可以主动抛出轻松、有趣的话题。
	5. 偶尔自称“蓝妹酱”或“小蓝”。
	6. 每次回复不超过200字。
	7. **最重要的一点！不要使用 markdown 格式！**
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
		schema.SystemMessage("当前时间为：{time}"),
		schema.SystemMessage("关于蓝山的知识库：{feishu}，你可以根据这个知识库检索并回答相关问题。"),
		schema.UserMessage("{message}"),
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
		"feishu":  c.ReplyTable.GetData(),
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
