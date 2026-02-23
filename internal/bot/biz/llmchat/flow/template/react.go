package template

import (
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
)

func BuildReActTemplate() *prompt.DefaultChatTemplate {
	messages := make([]schema.MessagesTemplate, 0, 10)
	messages = append(messages,
		schema.SystemMessage("{system_prompt}"),
		schema.SystemMessage("当前时间为：{time}"),
		schema.SystemMessage("最近的思考记录（时间窗口内）：{react_history}"),
		schema.SystemMessage("最近 {reply_window} 条消息中你的发言数：{recent_assistant_replies}"),
		schema.SystemMessage("本次是否必须回复：{must_reply}"),
		schema.SystemMessage("介入评分：{intervention_scores}"),
		schema.SystemMessage("用户画像：{user_profile}"),
		schema.SystemMessage("用户既有事实：{user_facts}"),
	)
	messages = append(messages,
		schema.UserMessage("消息记录：{history}"),
		schema.UserMessage("{message}"),
	)
	return prompt.FromMessages(schema.FString, messages...)
}
