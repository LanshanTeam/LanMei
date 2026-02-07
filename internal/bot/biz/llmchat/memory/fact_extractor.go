package memory

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	"LanMei/internal/bot/biz/llmchat/flow/hooks"
	"LanMei/internal/bot/utils/llog"

	fmodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
)

type FactExtractor struct {
	model    fmodel.ToolCallingChatModel
	template *prompt.DefaultChatTemplate
	hooks    *hooks.Runner
	hookInfo hooks.CallInfo
}

func NewFactExtractor(model fmodel.ToolCallingChatModel, hookRunner *hooks.Runner, hookInfo hooks.CallInfo) *FactExtractor {
	template := prompt.FromMessages(schema.FString,
		schema.SystemMessage("你是群聊事实抽取器，必须调用工具 extract_facts 输出参数，不要输出其他文本。"),
		schema.SystemMessage("任务：从聊天记录中提取稳定、可复用的既有事实（偏好/关系/经历/身份/长期状态）。不要记录瞬时情绪、一次性事件或不确定信息。"),
		schema.SystemMessage("事实必须以【某用户的事实】为中心，subject 填用户名字（昵称即可），content 用一句话描述事实。"),
		schema.SystemMessage("当信息不足以构成事实时，sufficient=false 且 facts 为空。"),
		schema.UserMessage("群:{group_id}"),
		schema.UserMessage("force:{force}"),
		schema.UserMessage("聊天记录:\n{events_text}"),
	)
	return &FactExtractor{
		model:    model,
		template: template,
		hooks:    hookRunner,
		hookInfo: hookInfo,
	}
}

func (e *FactExtractor) ExtractBatch(ctx context.Context, groupID string, messages []MemoryMessage, force bool) FactExtraction {
	if e == nil || e.model == nil || e.template == nil {
		return FactExtraction{}
	}
	if len(messages) == 0 {
		return FactExtraction{}
	}
	in, err := e.template.Format(ctx, map[string]any{
		"group_id":    groupID,
		"force":       force,
		"events_text": formatMemoryMessages(messages),
	})
	if err != nil {
		llog.Errorf("format fact extractor prompt error: %v", err)
		return FactExtraction{}
	}
	msg, err := hooks.Run(ctx, e.hooks, e.hookInfo, func() (*schema.Message, error) {
		return e.model.Generate(ctx, in)
	})
	if err != nil {
		llog.Errorf("generate fact extraction error: %v", err)
		return FactExtraction{}
	}
	for _, tc := range msg.ToolCalls {
		if tc.Function.Name != "extract_facts" {
			continue
		}
		var result FactExtraction
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &result); err != nil {
			llog.Errorf("解析事实抽取工具参数失败: %v", err)
			break
		}
		cleaned := make([]Fact, 0, len(result.Facts))
		for _, fact := range result.Facts {
			subject := strings.TrimSpace(fact.Subject)
			content := strings.TrimSpace(fact.Content)
			if subject == "" || content == "" {
				continue
			}
			fact.Subject = subject
			fact.Content = content
			cleaned = append(cleaned, fact)
		}
		result.Facts = cleaned
		return result
	}
	return FactExtraction{}
}

func BuildFactTool() *schema.ToolInfo {
	return &schema.ToolInfo{
		Name: "extract_facts",
		Desc: "抽取群聊中的稳定事实",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"sufficient": {
				Type:     schema.Boolean,
				Desc:     "当前记录是否足以抽取事实",
				Required: true,
			},
			"facts": {
				Type:     schema.Array,
				Desc:     "事实列表",
				Required: true,
				ElemInfo: &schema.ParameterInfo{
					Type: schema.Object,
					SubParams: map[string]*schema.ParameterInfo{
						"subject": {
							Type:     schema.String,
							Desc:     "用户名字",
							Required: true,
						},
						"content": {
							Type:     schema.String,
							Desc:     "事实内容",
							Required: true,
						},
						"confidence": {
							Type:     schema.Number,
							Desc:     "0-1 置信度",
							Required: true,
						},
					},
				},
			},
		}),
	}
}

func formatMemoryMessages(messages []MemoryMessage) string {
	lines := make([]string, 0, len(messages))
	for i, msg := range messages {
		speaker := messageSpeaker(msg)
		if speaker == "" {
			speaker = "用户"
		}
		line := strings.TrimSpace(msg.Content)
		lines = append(lines, formatMemoryLine(i+1, speaker, msg.Role, line))
	}
	return strings.Join(lines, "\n")
}

func formatMemoryLine(index int, speaker string, role schema.RoleType, content string) string {
	return "[" + strconv.Itoa(index) + "] " + speaker + "(" + string(role) + ") " + content
}

func messageSpeaker(msg MemoryMessage) string {
	if msg.Role == schema.Assistant {
		return "蓝妹"
	}
	if msg.Nickname != "" {
		return msg.Nickname
	}
	return msg.UserID
}
