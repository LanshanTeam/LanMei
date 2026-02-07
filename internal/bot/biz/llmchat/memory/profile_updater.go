package memory

import (
	"context"
	"encoding/json"
	"strings"

	"LanMei/internal/bot/biz/llmchat/flow/hooks"
	"LanMei/internal/bot/utils/llog"

	fmodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
)

type ProfileResult struct {
	Summary string   `json:"summary"`
	Tags    []string `json:"tags"`
}

type ProfileUpdater struct {
	model    fmodel.ToolCallingChatModel
	template *prompt.DefaultChatTemplate
	hooks    *hooks.Runner
	hookInfo hooks.CallInfo
}

func NewProfileUpdater(model fmodel.ToolCallingChatModel, hookRunner *hooks.Runner, hookInfo hooks.CallInfo) *ProfileUpdater {
	template := prompt.FromMessages(schema.FString,
		schema.SystemMessage("你是用户画像更新器，必须调用工具 build_user_profile 输出参数，不要输出其他文本。"),
		schema.SystemMessage("任务：根据既有事实更新用户画像，保持简洁、稳定，避免臆测。"),
		schema.SystemMessage("输出 summary 需包含以下小节：身份/关系/偏好/习惯/禁忌/常聊话题。"),
		schema.SystemMessage("summary 行格式：`身份:...` `关系:...` 等，要求尽量详细，为之后对话做准备。只写事实，不推测。"),
		schema.SystemMessage("summary 必须使用第三人称“他”描述，不要出现姓名/昵称/ID。"),
		schema.SystemMessage("tags 输出 5-12 个关键词，来自事实内容，避免泛化词。"),
		schema.UserMessage("用户ID:{subject}"),
		schema.UserMessage("已有画像:{current_profile}"),
		schema.UserMessage("既有事实列表:\n{facts}"),
	)
	return &ProfileUpdater{model: model, template: template, hooks: hookRunner, hookInfo: hookInfo}
}

func (u *ProfileUpdater) Update(ctx context.Context, subject string, facts []string, current ProfileResult) ProfileResult {
	if u == nil || u.model == nil || u.template == nil {
		return ProfileResult{}
	}
	factsText := strings.Join(facts, "\n")
	if strings.TrimSpace(factsText) == "" {
		factsText = "无"
	}
	in, err := u.template.Format(ctx, map[string]any{
		"subject":         subject,
		"current_profile": strings.TrimSpace(current.Summary),
		"facts":           factsText,
	})
	if err != nil {
		llog.Errorf("format profile prompt error: %v", err)
		return ProfileResult{}
	}
	msg, err := hooks.Run(ctx, u.hooks, u.hookInfo, func() (*schema.Message, error) {
		return u.model.Generate(ctx, in)
	})
	if err != nil {
		llog.Errorf("generate profile update error: %v", err)
		return ProfileResult{}
	}
	for _, tc := range msg.ToolCalls {
		if tc.Function.Name != "build_user_profile" {
			continue
		}
		var result ProfileResult
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &result); err != nil {
			llog.Errorf("解析用户画像工具参数失败: %v", err)
			break
		}
		return result
	}
	return ProfileResult{}
}

func BuildProfileTool() *schema.ToolInfo {
	return &schema.ToolInfo{
		Name: "build_user_profile",
		Desc: "构建用户画像摘要",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"summary": {
				Type:     schema.String,
				Desc:     "详细的画像摘要",
				Required: true,
			},
			"tags": {
				Type:     schema.Array,
				Desc:     "画像标签",
				Required: true,
				ElemInfo: &schema.ParameterInfo{Type: schema.String},
			},
		}),
	}
}
