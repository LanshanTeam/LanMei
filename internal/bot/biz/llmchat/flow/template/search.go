package template

import (
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
)

func BuildSearchFormatTemplate() *prompt.DefaultChatTemplate {
	rules := `你是“网络搜索结果内容提取与格式化器”。任务：把原始搜索结果整理为可读、可直接引用的要点，供上游模型使用。

【硬规则】
- 只输出整理后的内容，不要解释过程。
- 输出中文。
- 只基于原始结果，不要臆测/扩写。
- 去重合并相同观点。
- 每条后面保留来源（URL 或域名），用括号标注。
- 若原始结果无法提取信息，输出“无”。
`
	return prompt.FromMessages(schema.FString,
		schema.SystemMessage("你必须仅输出整理后的内容。"),
		schema.SystemMessage(rules),
		schema.UserMessage("原始输入（可以根据原始输入抓住查询的重点）：{input}"),
		schema.UserMessage("搜索关键词：{queries}"),
		schema.UserMessage("原始结果：{raw_results}"),
	)
}
