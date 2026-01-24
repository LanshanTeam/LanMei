package llmchat

import (
	"LanMei/internal/bot/utils/llog"
	"context"
	"encoding/json"
	"strings"

	"github.com/cloudwego/eino/schema"
)

type PlanResult struct {
	Action        string  `json:"action"`
	Intent        string  `json:"intent"`
	ReplyStyle    string  `json:"reply_style"`
	NeedMemory    bool    `json:"need_memory"`
	NeedKnowledge bool    `json:"need_knowledge"`
	NeedClarify   bool    `json:"need_clarify"`
	Confidence    float64 `json:"confidence"`
}

func (c *ChatEngine) buildPlan(ctx context.Context, nickname, input string, history []schema.Message) PlanResult {
	if c.planTemplate == nil || c.plannerModel == nil {
		return PlanResult{}
	}
	in, err := c.planTemplate.Format(ctx, map[string]any{
		"nickname": nickname,
		"history":  history,
		"message":  input,
	})
	if err != nil {
		llog.Error("format plan message error: %v", err)
		return PlanResult{}
	}
	msg, err := c.plannerModel.Generate(ctx, in)
	if err != nil {
		llog.Error("generate plan error: %v", err)
		return PlanResult{}
	}
	for _, tc := range msg.ToolCalls {
		if tc.Function.Name != "plan_chat" {
			continue
		}
		var plan PlanResult
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &plan); err != nil {
			llog.Error("解析 planner tool 参数失败: %v", err)
			break
		}
		plan.Intent = strings.TrimSpace(plan.Intent)
		return plan
	}
	return PlanResult{}
}
