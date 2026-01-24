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

type InputAnalysis struct {
	RawInput        string   `json:"-"`
	OptimizedInput  string   `json:"optimized_input"`
	Intent          string   `json:"intent"`
	Purpose         string   `json:"purpose"`
	PsychState      string   `json:"psych_state"`
	SlangTerms      []string `json:"slang_terms"`
	UnknownTerms    []string `json:"unknown_terms"`
	AddressedTarget string   `json:"addressed_target"`
	TargetDetail    string   `json:"target_detail"`
	NeedClarify     bool     `json:"need_clarify"`
}

type InputAnalyzer struct {
	model    fmodel.ToolCallingChatModel
	template *prompt.DefaultChatTemplate
}

func NewInputAnalyzer(model fmodel.ToolCallingChatModel) *InputAnalyzer {
	template := prompt.FromMessages(schema.FString,
		schema.SystemMessage("你是输入分析器，必须调用工具 analyze_input 输出参数，不要输出其他文本。"),
		schema.SystemMessage("optimized_input 用于检索与规划，应简洁清晰，保留关键信息。"),
		schema.SystemMessage("intent 是一句话概括，purpose 是更深层的说话目的。psych_state 描述用户可能的心理/情绪活动。"),
		schema.SystemMessage("slang_terms 列出用户话里的俚语/梗（即使你理解，可为空）；unknown_terms 仅包含你不理解、可能需要记录的词语/俚语，可为空，必要时可与 slang_terms 重叠。"),
		schema.SystemMessage("addressed_target 只能是 me|other|group|unknown；target_detail 仅在 other/group 时填写具体对象，否则填 无。"),
		schema.UserMessage("用户昵称：{nickname}"),
		schema.UserMessage("最近消息：{history}"),
		schema.UserMessage("当前消息：{message}"),
	)
	return &InputAnalyzer{
		model:    model,
		template: template,
	}
}

func (a *InputAnalyzer) Analyze(ctx context.Context, nickname, input string, history []schema.Message) (InputAnalysis, bool) {
	if a == nil || a.model == nil || a.template == nil {
		return InputAnalysis{}, false
	}
	in, err := a.template.Format(ctx, map[string]any{
		"nickname": nickname,
		"history":  history,
		"message":  input,
	})
	if err != nil {
		llog.Error("format input analysis error: %v", err)
		return InputAnalysis{}, false
	}
	msg, err := a.model.Generate(ctx, in)
	if err != nil {
		llog.Error("generate input analysis error: %v", err)
		return InputAnalysis{}, false
	}
	for _, tc := range msg.ToolCalls {
		if tc.Function.Name != "analyze_input" {
			continue
		}
		var analysis InputAnalysis
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &analysis); err != nil {
			llog.Error("解析 analyze_input 参数失败: %v", err)
			break
		}
		analysis.RawInput = input
		analysis.OptimizedInput = strings.TrimSpace(analysis.OptimizedInput)
		analysis.Intent = strings.TrimSpace(analysis.Intent)
		analysis.Purpose = strings.TrimSpace(analysis.Purpose)
		analysis.PsychState = strings.TrimSpace(analysis.PsychState)
		analysis.AddressedTarget = strings.TrimSpace(analysis.AddressedTarget)
		analysis.TargetDetail = strings.TrimSpace(analysis.TargetDetail)
		return analysis, true
	}
	return InputAnalysis{}, false
}
