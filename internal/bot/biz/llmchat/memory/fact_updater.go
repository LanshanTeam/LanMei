package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"LanMei/internal/bot/biz/llmchat/hooks"
	"LanMei/internal/bot/utils/llog"

	fmodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
)

type FactUpdater struct {
	model    fmodel.ToolCallingChatModel
	template *prompt.DefaultChatTemplate
	hooks    *hooks.Runner
	hookInfo hooks.CallInfo
}

func NewFactUpdater(model fmodel.ToolCallingChatModel, hookRunner *hooks.Runner, hookInfo hooks.CallInfo) *FactUpdater {
	template := prompt.FromMessages(schema.FString,
		schema.SystemMessage("你是事实更新决策器，必须调用工具 apply_fact_update 输出参数，不要输出其他文本。"),
		schema.SystemMessage("任务：给定新事实与相似旧事实，决定 ADD/UPDATE/DELETE/NONE。更新用于纠正矛盾或更精确事实；删除用于否定旧事实；不确定则 NONE。"),
		schema.SystemMessage("输出 event 只能是 ADD|UPDATE|DELETE|NONE；UPDATE/DELETE 必须给 target_id。"),
		schema.UserMessage("用户:{subject}"),
		schema.UserMessage("新事实:{new_fact}"),
		schema.UserMessage("相似旧事实列表(含ID):\n{old_facts}"),
	)
	return &FactUpdater{
		model:    model,
		template: template,
		hooks:    hookRunner,
		hookInfo: hookInfo,
	}
}

func (u *FactUpdater) Decide(ctx context.Context, subject, newFact string, oldFacts []FactRecord) FactDecision {
	if u == nil || u.model == nil || u.template == nil {
		return FactDecision{Event: "ADD", Text: newFact}
	}
	oldFactsText := formatFactRecords(oldFacts)
	in, err := u.template.Format(ctx, map[string]any{
		"subject":   subject,
		"new_fact":  newFact,
		"old_facts": oldFactsText,
	})
	if err != nil {
		llog.Errorf("format fact update prompt error: %v", err)
		return FactDecision{Event: "ADD", Text: newFact}
	}
	msg, err := hooks.Run(ctx, u.hooks, u.hookInfo, func() (*schema.Message, error) {
		return u.model.Generate(ctx, in)
	})
	if err != nil {
		llog.Errorf("generate fact update error: %v", err)
		return FactDecision{Event: "ADD", Text: newFact}
	}
	for _, tc := range msg.ToolCalls {
		if tc.Function.Name != "apply_fact_update" {
			continue
		}
		var decision FactDecision
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &decision); err != nil {
			llog.Errorf("解析事实更新工具参数失败: %v", err)
			break
		}
		decision.Event = strings.ToUpper(strings.TrimSpace(decision.Event))
		decision.Text = strings.TrimSpace(decision.Text)
		return decision
	}
	return FactDecision{Event: "ADD", Text: newFact}
}

func BuildFactUpdateTool() *schema.ToolInfo {
	return &schema.ToolInfo{
		Name: "apply_fact_update",
		Desc: "根据新事实与旧事实决定更新策略",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"event": {
				Type:     schema.String,
				Desc:     "ADD|UPDATE|DELETE|NONE",
				Required: true,
			},
			"text": {
				Type:     schema.String,
				Desc:     "用于 ADD/UPDATE 的新事实文本",
				Required: true,
			},
			"target_id": {
				Type:     schema.String,
				Desc:     "UPDATE/DELETE 对应的旧事实ID",
				Required: false,
			},
			"reason": {
				Type:     schema.String,
				Desc:     "更新理由(可选)",
				Required: false,
			},
		}),
	}
}

func formatFactRecords(records []FactRecord) string {
	if len(records) == 0 {
		return "无"
	}
	lines := make([]string, 0, len(records))
	for _, r := range records {
		lines = append(lines, fmt.Sprintf("ID:%d 内容:%s", r.ID, strings.TrimSpace(r.Text)))
	}
	return strings.Join(lines, "\n")
}
