package flow

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"LanMei/internal/bot/biz/dao"
	"LanMei/internal/bot/biz/llmchat/analysis"
	"LanMei/internal/bot/biz/llmchat/hooks"
	"LanMei/internal/bot/biz/llmchat/memory"
	"LanMei/internal/bot/utils/llog"
	"LanMei/internal/bot/utils/sensitive"

	"github.com/cloudwego/eino/schema"
)

func initNode(deps Dependencies) func(context.Context, *State) (*State, error) {
	return func(ctx context.Context, state *State) (*State, error) {
		if state == nil {
			return state, nil
		}
		state.History = loadAndStoreHistory(deps.Memory, state.Request.GroupID, state.Request.UserID, state.Request.Nickname, state.Request.Input)
		if !state.Request.Must && deps.Frequency != nil && deps.Frequency.ShouldThrottle(state.Request.GroupID) {
			llog.Info("频率控制，不回复")
			state.StopWith("throttle")
			return state, nil
		}
		return state, nil
	}
}

func userContextNode(deps Dependencies) func(context.Context, *State) (*State, error) {
	return func(ctx context.Context, state *State) (*State, error) {
		if state == nil || state.Stop {
			return state, nil
		}
		if deps.Memory == nil {
			state.UserFacts = nil
			state.UserProfile = "无"
			return state, nil
		}
		name := strings.TrimSpace(state.Request.Nickname)
		if name == "" {
			name = strings.TrimSpace(state.Request.UserID)
		}
		facts, profile := deps.Memory.GetUserContext(ctx, state.Request.GroupID, name, 12)
		state.UserFacts = facts
		state.UserProfile = profile
		return state, nil
	}
}

func analysisNode(deps Dependencies) func(context.Context, *State) (*State, error) {
	return func(ctx context.Context, state *State) (*State, error) {
		if state == nil || state.Stop {
			return state, nil
		}
		if deps.InputAnalyzer == nil {
			state.StopWith("analysis_unavailable")
			return state, nil
		}
		analysisResult, ok := deps.InputAnalyzer.Analyze(ctx, state.Request.Nickname, state.Request.Input, state.History, state.UserFacts, state.UserProfile)
		if !ok {
			state.StopWith("analysis_failed")
			return state, nil
		}
		analysisResult = analysis.Normalize(analysisResult, state.Request.Input)
		llog.Info("意图分析：", analysisResult)
		state.Analysis = analysisResult
		return state, nil
	}
}

func judgeNode(deps Dependencies) func(context.Context, *State) (*State, error) {
	return func(ctx context.Context, state *State) (*State, error) {
		if state == nil || state.Stop {
			return state, nil
		}
		if state.Request.Must {
			return state, nil
		}
		if deps.JudgeModel == nil || deps.JudgeTemplate == nil {
			state.StopWith("judge_unavailable")
			return state, nil
		}
		recentAssistantReplies := recentAssistantReplies(state.History, replyFrequencyWindow)
		judgeIn, err := deps.JudgeTemplate.Format(ctx, map[string]any{
			"message":                  state.Request.Input,
			"history":                  state.History,
			"intent":                   state.Analysis.Intent,
			"purpose":                  state.Analysis.Purpose,
			"psych_state":              state.Analysis.PsychState,
			"addressed_target":         state.Analysis.AddressedTarget,
			"target_detail":            state.Analysis.TargetDetail,
			"optimized_input":          state.Analysis.OptimizedInput,
			"recent_assistant_replies": recentAssistantReplies,
			"reply_window":             replyFrequencyWindow,
		})
		if err != nil {
			llog.Error("format judge message error: %v", err)
			state.StopWith("judge_format_error")
			return state, nil
		}
		judgeMsg, err := hooks.Run(ctx, deps.Hooks, deps.HookInfos.Judge, func() (*schema.Message, error) {
			return deps.JudgeModel.Generate(ctx, judgeIn)
		})
		if err != nil {
			llog.Error("generate judge message error: %v", err)
			state.StopWith("judge_error")
			return state, nil
		}
		if len(judgeMsg.ToolCalls) == 0 {
			state.StopWith("judge_no_tool")
			return state, nil
		}
		for _, tc := range judgeMsg.ToolCalls {
			if tc.Function.Name != "interested_scores" {
				continue
			}
			var params map[string]interface{}
			if err := json.Unmarshal([]byte(tc.Function.Arguments), &params); err != nil {
				llog.Error("unmarshal arguments error: %v", err)
				state.StopWith("judge_unmarshal_error")
				return state, nil
			}
			score, ok := computeReplyScore(params)
			if !ok {
				state.StopWith("judge_score_blocked")
				return state, nil
			}
			repeatPenalty := clampPenalty(toFloat(params["repeat_penalty"]))
			frequencyPenalty := clampPenalty(toFloat(params["frequency_penalty"]))
			penalty := repeatPenalty + frequencyPenalty
			if penalty > replyPenaltyMax {
				penalty = replyPenaltyMax
			}
			threshold := baseReplyScoreThreshold + penalty
			llog.Info(fmt.Sprintf("should Reply: params=%v score=%.1f penalty=%.1f threshold=%.1f", params, score, penalty, threshold))
			if score >= threshold {
				return state, nil
			}
			state.StopWith("judge_threshold_blocked")
			return state, nil
		}
		state.StopWith("judge_no_score")
		return state, nil
	}
}

func planNode(deps Dependencies) func(context.Context, *State) (*State, error) {
	return func(ctx context.Context, state *State) (*State, error) {
		if state == nil || state.Stop {
			return state, nil
		}
		plan := buildPlan(ctx, deps, state)
		if plan.Action == "" {
			state.StopWith("plan_empty")
			return state, nil
		}
		if plan.Action == "wait" && !state.Request.Must {
			state.StopWith("plan_wait")
			return state, nil
		}
		if plan.Action == "ask_clarify" {
			plan.NeedClarify = true
		}
		state.Plan = plan
		return state, nil
	}
}

func gatherContextNode(deps Dependencies) func(context.Context, *State) (*State, error) {
	return func(ctx context.Context, state *State) (*State, error) {
		if state == nil || state.Stop {
			return state, nil
		}
		state.MemoryBlock = recallMemory(ctx, deps.Memory, state.Analysis.OptimizedInput, state.Request.GroupID, state.Plan.NeedMemory)
		state.Knowledge = recallKnowledge(ctx, deps, state.Analysis.OptimizedInput, state.Plan.NeedKnowledge)
		state.WebSearchRaw = recallWebSearchRaw(ctx, deps, state.Analysis)
		return state, nil
	}
}

func searchFormatNode(deps Dependencies) func(context.Context, *State) (*State, error) {
	return func(ctx context.Context, state *State) (*State, error) {
		if state == nil || state.Stop {
			return state, nil
		}
		state.WebSearch = formatSearchResults(ctx, deps, state.Analysis.RawInput, state.Analysis.SearchQueries, state.WebSearchRaw)
		return state, nil
	}
}

func buildPromptNode(deps Dependencies) func(context.Context, *State) (*State, error) {
	return func(ctx context.Context, state *State) (*State, error) {
		if state == nil || state.Stop {
			return state, nil
		}
		if deps.Template == nil {
			state.StopWith("template_missing")
			return state, nil
		}
		rawInput := strings.TrimSpace(state.Analysis.RawInput)
		if rawInput == "" {
			rawInput = state.Analysis.OptimizedInput
		}
		augmentedInput := state.Request.Nickname + "说：" + rawInput
		promptInput, err := deps.Template.Format(ctx, map[string]any{
			"message":          augmentedInput,
			"time":             time.Now(),
			"feishu":           state.Knowledge,
			"history":          state.History,
			"memory":           state.MemoryBlock,
			"web_search":       state.WebSearch,
			"user_profile":     formatUserProfile(state.UserProfile),
			"user_facts":       formatUserFacts(state.UserFacts),
			"plan":             formatPlan(state.Plan),
			"intent":           state.Analysis.Intent,
			"purpose":          state.Analysis.Purpose,
			"psych_state":      state.Analysis.PsychState,
			"addressed_target": state.Analysis.AddressedTarget,
			"target_detail":    state.Analysis.TargetDetail,
			"raw_input":        rawInput,
			"optimized_input":  state.Analysis.OptimizedInput,
			"reply_style":      state.Plan.ReplyStyle,
		})
		if err != nil {
			llog.Error("format message error: %v", err)
			state.Reply = state.Request.Input
			state.StopWith("prompt_format_error")
			return state, nil
		}
		state.Prompt = promptInput
		return state, nil
	}
}

func chatNode(deps Dependencies) func(context.Context, *State) (*State, error) {
	return func(ctx context.Context, state *State) (*State, error) {
		if state == nil || state.Stop {
			return state, nil
		}
		if deps.ChatModel == nil {
			state.StopWith("chat_unavailable")
			return state, nil
		}
		msg, err := hooks.Run(ctx, deps.Hooks, deps.HookInfos.Chat, func() (*schema.Message, error) {
			return deps.ChatModel.Generate(ctx, state.Prompt)
		})
		if err != nil {
			llog.Error("generate message error: %v", err)
			state.Reply = state.Request.Input
			state.StopWith("chat_generate_error")
			return state, nil
		}
		llog.Info("消耗 Completion Tokens: ", msg.ResponseMeta.Usage.CompletionTokens)
		llog.Info("消耗 Prompt Tokens: ", msg.ResponseMeta.Usage.PromptTokens)
		llog.Info("消耗 Total Tokens: ", msg.ResponseMeta.Usage.TotalTokens)
		llog.Info("输出消息为：", msg.Content)
		state.Reply = msg.Content
		return state, nil
	}
}

func postProcessNode(deps Dependencies) func(context.Context, *State) (*State, error) {
	return func(ctx context.Context, state *State) (*State, error) {
		if state == nil || state.Stop {
			return state, nil
		}
		if sensitive.HaveSensitive(state.Reply) {
			state.Reply = "唔唔~小蓝的数据库里没有这种词哦，要不要换个萌萌的说法呀~(>ω<)"
			state.StopWith("sensitive_blocked")
			return state, nil
		}
		if deps.Memory != nil {
			deps.Memory.AppendAssistant(state.Request.GroupID, state.Reply)
		}
		if deps.Frequency != nil {
			deps.Frequency.MarkSent(state.Request.GroupID)
		}
		return state, nil
	}
}

func loadAndStoreHistory(memoryManager *memory.MemoryManager, groupID, userID, nickname, input string) []schema.Message {
	if memoryManager == nil {
		return []schema.Message{}
	}
	return memoryManager.LoadHistoryAndAppendUser(groupID, userID, nickname, input)
}

func recallMemory(ctx context.Context, memoryManager *memory.MemoryManager, query, groupID string, needMemory bool) string {
	if memoryManager == nil || !needMemory {
		return "无"
	}
	memorySnippets := memoryManager.Retrieve(ctx, query, groupID, needMemory)
	if len(memorySnippets) == 0 {
		return "无"
	}
	return strings.Join(memorySnippets, "\n")
}

func recallKnowledge(ctx context.Context, deps Dependencies, query string, needKnowledge bool) []string {
	if !needKnowledge {
		return nil
	}
	if dao.DBManager == nil {
		return nil
	}
	msgs := dao.DBManager.GetTopK(ctx, dao.CollectionName, 50, query)
	if deps.Reranker != nil {
		reranked := deps.Reranker.TopN(8, msgs, query)
		if len(reranked) > 0 {
			msgs = reranked
		}
	}
	return msgs
}

func recallWebSearchRaw(ctx context.Context, deps Dependencies, input analysis.InputAnalysis) string {
	if deps.Searcher == nil || !input.NeedSearch {
		return "无"
	}
	queries := append([]string(nil), input.SearchQueries...)
	if len(queries) == 0 && strings.TrimSpace(input.OptimizedInput) != "" {
		queries = []string{strings.TrimSpace(input.OptimizedInput)}
	}
	maxQueries := 3
	if len(queries) > maxQueries {
		queries = queries[:maxQueries]
	}
	blocks := make([]string, 0, len(queries))
	for _, query := range queries {
		results, err := deps.Searcher.Search(ctx, query, 4)
		if err != nil {
			llog.Error("网络检索失败: %v", err)
			continue
		}
		block := formatWebSearch(results)
		if block == "无" {
			continue
		}
		blocks = append(blocks, fmt.Sprintf("查询:%s -> 获取结果为：%s \n", query, block))
	}
	llog.Info("网络搜索结果：", blocks)
	if len(blocks) == 0 {
		return "无"
	}
	return strings.Join(blocks, "\n")
}

func formatSearchResults(ctx context.Context, deps Dependencies, input string, queries []string, raw string) string {
	if deps.SearchModel == nil || deps.SearchTemplate == nil {
		return raw
	}
	if strings.TrimSpace(raw) == "" || strings.TrimSpace(raw) == "无" {
		return raw
	}
	in, err := deps.SearchTemplate.Format(ctx, map[string]any{
		"input":       input,
		"queries":     strings.Join(queries, "、"),
		"raw_results": raw,
	})
	if err != nil {
		llog.Error("format search summary error: %v", err)
		return raw
	}
	msg, err := hooks.Run(ctx, deps.Hooks, deps.HookInfos.Search, func() (*schema.Message, error) {
		return deps.SearchModel.Generate(ctx, in)
	})
	if err != nil {
		llog.Error("generate search summary error: %v", err)
		return raw
	}
	content := strings.TrimSpace(msg.Content)
	if content == "" {
		return raw
	}
	llog.Info("格式化后的结果", content)
	return content
}

func formatUserFacts(facts []string) string {
	if len(facts) == 0 {
		return "无"
	}
	return strings.Join(facts, "\n")
}

func formatUserProfile(profile string) string {
	profile = strings.TrimSpace(profile)
	if profile == "" {
		return "无"
	}
	return profile
}
