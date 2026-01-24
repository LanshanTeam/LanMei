package llmchat

import (
	"LanMei/internal/bot/utils/llog"
	"context"
	"encoding/json"
	"strings"

	fmodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
)

type JargonMeaning struct {
	Term    string `json:"term"`
	Meaning string `json:"meaning"`
	NoInfo  bool   `json:"no_info"`
}

type JargonLearner struct {
	model    fmodel.ToolCallingChatModel
	template *prompt.DefaultChatTemplate
}

func NewJargonLearner(model fmodel.ToolCallingChatModel) *JargonLearner {
	template := prompt.FromMessages(schema.FString,
		schema.SystemMessage("你是俚语解释器，必须调用工具 infer_jargon 输出参数，不要输出其他文本。"),
		schema.SystemMessage("若信息不足无法确定含义，将 no_info 设置为 true。"),
		schema.UserMessage("词条:{term}"),
		schema.UserMessage("历史上下文:{contexts}"),
		schema.UserMessage("上下文:{context}"),
	)
	return &JargonLearner{
		model:    model,
		template: template,
	}
}

func (l *JargonLearner) Infer(ctx context.Context, term string, contexts []string, contextText string) (JargonMeaning, bool) {
	if l == nil || l.model == nil || l.template == nil {
		return JargonMeaning{}, false
	}
	mergedContexts := strings.Join(contexts, "\n")
	if mergedContexts == "" {
		mergedContexts = "无"
	}
	in, err := l.template.Format(ctx, map[string]any{
		"term":     term,
		"contexts": mergedContexts,
		"context":  contextText,
	})
	if err != nil {
		llog.Error("format jargon inference prompt error: %v", err)
		return JargonMeaning{}, false
	}
	msg, err := l.model.Generate(ctx, in)
	if err != nil {
		llog.Error("generate jargon inference error: %v", err)
		return JargonMeaning{}, false
	}
	for _, tc := range msg.ToolCalls {
		if tc.Function.Name != "infer_jargon" {
			continue
		}
		var result JargonMeaning
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &result); err != nil {
			llog.Error("解析俚语推断工具参数失败: %v", err)
			break
		}
		result.Term = strings.TrimSpace(result.Term)
		result.Meaning = strings.TrimSpace(result.Meaning)
		return result, true
	}
	return JargonMeaning{}, false
}
