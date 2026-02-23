package nodes

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	flowtypes "LanMei/internal/bot/biz/llmchat/flow/types"
	reactbiz "LanMei/internal/bot/biz/llmchat/react"
	"LanMei/internal/bot/biz/llmchat/reactlog"
	"LanMei/internal/bot/utils/llog"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent"
	"github.com/cloudwego/eino/schema"
	cb "github.com/cloudwego/eino/utils/callbacks"
)

const reactHistoryMaxChars = 1600

const reactBasePrompt = `你是蓝妹的 ReAct 对话执行器。
- 先调用 list_skills/view_skill 获取并遵守：lanmei_persona、group_chat_analysis、group_chat_planner、group_chat_judge、search_formatter、react_core。
- 需要工具时再调用工具；完成后必须调用 final_response 输出 action 和 content。
- 不要在最终回复中暴露 Thought/Action/Observation。`

type finalResponsePayload struct {
	Action  string `json:"action"`
	Content string `json:"content"`
}

func ReActNode(deps flowtypes.Dependencies) func(context.Context, *flowtypes.State) (*flowtypes.State, error) {
	return func(ctx context.Context, state *flowtypes.State) (*flowtypes.State, error) {
		if state == nil || state.Stop {
			return state, nil
		}
		if deps.ReActAgent == nil || deps.ReActTemplate == nil {
			state.StopWith("react_unavailable")
			return state, nil
		}
		rawInput := strings.TrimSpace(state.Request.Input)
		if rawInput == "" {
			state.StopWith("react_empty_input")
			return state, nil
		}
		recentReplies := recentAssistantReplies(state.History, replyFrequencyWindow)
		reactHistory := "无"
		if deps.ReActLog != nil {
			reactHistory = deps.ReActLog.Format(state.Request.GroupID, reactHistoryMaxChars)
		}
		systemPrompt := reactBasePrompt
		if deps.SkillPromptInjector != nil {
			systemPrompt = deps.SkillPromptInjector(systemPrompt)
		}
		message := state.Request.Nickname + "说：" + rawInput
		prompt, err := deps.ReActTemplate.Format(ctx, map[string]any{
			"system_prompt":            systemPrompt,
			"time":                     time.Now(),
			"react_history":            reactHistory,
			"reply_window":             replyFrequencyWindow,
			"recent_assistant_replies": recentReplies,
			"must_reply":               state.Request.Must,
			"intervention_scores":      formatInterventionScores(state.Request.Intervention),
			"user_profile":             state.UserProfile,
			"user_facts":               formatUserFacts(state.UserFacts),
			"history":                  state.History,
			"message":                  message,
		})
		if err != nil {
			llog.Error("format react prompt error: %v", err)
			state.StopWith("react_prompt_error")
			return state, nil
		}
		ctx = reactlog.ContextWithGroupID(ctx, state.Request.GroupID)
		ctx = reactlog.ContextWithQuery(ctx, rawInput)
		traceHandler := buildReActTraceHandler(deps.ReActLog)
		var msg *schema.Message
		if traceHandler == nil {
			msg, err = deps.ReActAgent.Generate(ctx, prompt)
		} else {
			msg, err = deps.ReActAgent.Generate(ctx, prompt, agent.WithComposeOptions(compose.WithCallbacks(traceHandler)))
		}
		if err != nil {
			llog.Error("react agent error: %v", err)
			state.StopWith("react_error")
			return state, nil
		}
		action, reply := parseReActResult(msg)
		if action == "wait" {
			if state.Request.Must {
				if strings.TrimSpace(reply) == "" {
					reply = "嗯？"
				}
			} else {
				state.StopWith("react_wait")
				return state, nil
			}
		}
		if strings.TrimSpace(reply) == "" {
			state.StopWith("react_empty_reply")
			return state, nil
		}
		state.Reply = reply
		if deps.ReActLog != nil {
			deps.ReActLog.Append(state.Request.GroupID, reactlog.StepFinal, reply)
		}
		return state, nil
	}
}

func parseReActResult(msg *schema.Message) (string, string) {
	if msg == nil {
		return "wait", ""
	}
	if msg.Role == schema.Tool {
		return parseFinalToolMessage(msg)
	}
	content := strings.TrimSpace(msg.Content)
	if content == "" {
		return "wait", ""
	}
	return "reply", content
}

func parseFinalToolMessage(msg *schema.Message) (string, string) {
	if msg == nil {
		return "wait", ""
	}
	if msg.ToolName != reactbiz.ToolFinalResponse {
		content := strings.TrimSpace(msg.Content)
		if content == "" {
			return "wait", ""
		}
		return "reply", content
	}
	var payload finalResponsePayload
	if err := json.Unmarshal([]byte(msg.Content), &payload); err != nil {
		llog.Error("解析 final_response 输出失败: %v", err)
		return "wait", ""
	}
	action := strings.TrimSpace(payload.Action)
	if action == "" {
		action = "reply"
	}
	return action, strings.TrimSpace(payload.Content)
}

func buildReActTraceHandler(trace *reactlog.Window) callbacks.Handler {
	if trace == nil {
		return nil
	}
	return cb.NewHandlerHelper().ChatModel(&cb.ModelCallbackHandler{
		OnEnd: func(ctx context.Context, _ *callbacks.RunInfo, output *model.CallbackOutput) context.Context {
			if output == nil || output.Message == nil {
				return ctx
			}
			groupID := reactlog.GroupIDFromContext(ctx)
			if groupID == "" {
				return ctx
			}
			msg := output.Message
			if strings.TrimSpace(msg.ReasoningContent) != "" {
				trace.Append(groupID, reactlog.StepThought, msg.ReasoningContent)
			}
			if strings.TrimSpace(msg.Content) != "" {
				trace.Append(groupID, reactlog.StepThought, msg.Content)
			}
			llog.Info("ReAct 执行步骤\n", msg)
			return ctx
		},
	}).Handler()
}

func formatUserFacts(facts []string) string {
	if len(facts) == 0 {
		return "无"
	}
	trimmed := make([]string, 0, len(facts))
	for _, fact := range facts {
		fact = strings.TrimSpace(fact)
		if fact == "" {
			continue
		}
		trimmed = append(trimmed, fact)
	}
	if len(trimmed) == 0 {
		return "无"
	}
	return strings.Join(trimmed, "\n")
}
