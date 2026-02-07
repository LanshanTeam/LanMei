package nodes

import (
	"context"
	"encoding/json"
	"strings"

	"LanMei/internal/bot/biz/llmchat/flow/hooks"
	llmtemplate "LanMei/internal/bot/biz/llmchat/flow/template"
	flowtypes "LanMei/internal/bot/biz/llmchat/flow/types"
	"LanMei/internal/bot/utils/llog"

	fmodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
)

type InputAnalyzer struct {
	model    fmodel.ToolCallingChatModel
	template *prompt.DefaultChatTemplate
	hooks    *hooks.Runner
	hookInfo hooks.CallInfo
}

func NewInputAnalyzer(model fmodel.ToolCallingChatModel, hookRunner *hooks.Runner, hookInfo hooks.CallInfo) *InputAnalyzer {
	template := llmtemplate.BuildInputAnalysisTemplate()
	return &InputAnalyzer{model: model, template: template, hooks: hookRunner, hookInfo: hookInfo}
}

func (a *InputAnalyzer) Analyze(ctx context.Context, nickname, input string, history []schema.Message, knownFacts []string, userProfile string) (flowtypes.InputAnalysis, bool) {
	if a == nil || a.model == nil || a.template == nil {
		return flowtypes.InputAnalysis{}, false
	}
	in, err := a.template.Format(ctx, map[string]any{
		"nickname":     nickname,
		"user_profile": normalizeUserProfile(userProfile),
		"known_facts":  normalizeKnownFacts(knownFacts),
		"history":      history,
		"message":      input,
	})
	if err != nil {
		llog.Error("format input analysis error: %v", err)
		return flowtypes.InputAnalysis{}, false
	}
	msg, err := hooks.Run(ctx, a.hooks, a.hookInfo, func() (*schema.Message, error) {
		return a.model.Generate(ctx, in)
	})
	if err != nil {
		llog.Errorf("generate input analysis error: %v", err)
		return flowtypes.InputAnalysis{}, false
	}
	for _, tc := range msg.ToolCalls {
		if tc.Function.Name != "analyze_input" {
			continue
		}
		var analysis flowtypes.InputAnalysis
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &analysis); err != nil {
			llog.Errorf("解析 analyze_input 参数失败: %v", err)
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
	return flowtypes.InputAnalysis{}, false
}

func BuildTool() *schema.ToolInfo {
	return &schema.ToolInfo{
		Name: "analyze_input",
		Desc: "根据当前消息与上下文生成输入优化与意图分析结果",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"optimized_input": {
				Type:     schema.String,
				Desc:     "优化后的输入，便于检索与规划",
				Required: true,
			},
			"intent": {
				Type:     schema.String,
				Desc:     "简短意图（一句话概括）",
				Required: true,
			},
			"purpose": {
				Type:     schema.String,
				Desc:     "更深层的说话目的（求关注/求安慰/分享/试探等）",
				Required: true,
			},
			"psych_state": {
				Type:     schema.String,
				Desc:     "用户可能的心理/情绪活动",
				Required: true,
			},
			"addressed_target": {
				Type:     schema.String,
				Desc:     "说话对象：me|other|group|unknown",
				Required: true,
			},
			"target_detail": {
				Type:     schema.String,
				Desc:     "当对象为 other/group 时的具体对象描述，否则填 无",
				Required: true,
			},
			"need_clarify": {
				Type:     schema.Boolean,
				Desc:     "是否需要澄清",
				Required: true,
			},
			"need_search": {
				Type:     schema.Boolean,
				Desc:     "是否需要网络搜索(地点/位置/事件/名词解释/新发布游戏/最新版本/技术前沿等)",
				Required: true,
			},
			"search_queries": {
				Type:     schema.Array,
				Desc:     "用于网络搜索的关键词数组，简短，可为空",
				Required: true,
				ElemInfo: &schema.ParameterInfo{
					Type: schema.String,
				},
			},
		}),
	}
}

func Normalize(analysis flowtypes.InputAnalysis, rawInput string) flowtypes.InputAnalysis {
	if analysis.RawInput == "" {
		analysis.RawInput = rawInput
	}
	if strings.TrimSpace(analysis.OptimizedInput) == "" {
		analysis.OptimizedInput = rawInput
	}
	analysis.OptimizedInput = strings.TrimSpace(analysis.OptimizedInput)
	if analysis.NeedSearch {
		normalized := make([]string, 0, len(analysis.SearchQueries))
		for _, query := range analysis.SearchQueries {
			query = strings.TrimSpace(query)
			if query == "" || query == "无" {
				continue
			}
			normalized = append(normalized, query)
		}
		if len(normalized) == 0 && strings.TrimSpace(analysis.OptimizedInput) != "" {
			normalized = append(normalized, strings.TrimSpace(analysis.OptimizedInput))
		}
		analysis.SearchQueries = dedupeStrings(normalized)
	}
	return analysis
}

func dedupeStrings(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	out := make([]string, 0, len(items))
	for _, item := range items {
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}

func normalizeUserProfile(profile string) string {
	profile = strings.TrimSpace(profile)
	if profile == "" {
		return "无"
	}
	return profile
}

func normalizeKnownFacts(facts []string) string {
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
