package llmchat

import (
	"LanMei/internal/bot/biz/llmchat/analysis"
	"LanMei/internal/bot/biz/llmchat/hooks"
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

func (c *ChatEngine) buildPlan(ctx context.Context, nickname string, analysisResult analysis.InputAnalysis, history []schema.Message) PlanResult {
	if c.planTemplate == nil || c.plannerModel == nil {
		return PlanResult{}
	}
	message := strings.TrimSpace(analysisResult.OptimizedInput)
	if message == "" {
		message = strings.TrimSpace(analysisResult.RawInput)
	}
	in, err := c.planTemplate.Format(ctx, map[string]any{
		"nickname":         nickname,
		"history":          history,
		"message":          message,
		"intent":           analysisResult.Intent,
		"purpose":          analysisResult.Purpose,
		"psych_state":      analysisResult.PsychState,
		"addressed_target": analysisResult.AddressedTarget,
		"target_detail":    analysisResult.TargetDetail,
		"optimized_input":  analysisResult.OptimizedInput,
	})
	if err != nil {
		llog.Error("format plan message error: %v", err)
		return PlanResult{}
	}
	msg, err := hooks.Run(ctx, c.hooks, c.planHookInfo, func() (*schema.Message, error) {
		return c.plannerModel.Generate(ctx, in)
	})
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
		llog.Info("plan: ", plan)
		plan.Intent = strings.TrimSpace(plan.Intent)
		return plan
	}
	return PlanResult{}
}
