package analysis

import (
	"LanMei/internal/bot/biz/llmchat/hooks"
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
	AddressedTarget string   `json:"addressed_target"`
	TargetDetail    string   `json:"target_detail"`
	NeedClarify     bool     `json:"need_clarify"`
	NeedSearch      bool     `json:"need_search"`
	SearchQueries   []string `json:"search_queries"`
}

type InputAnalyzer struct {
	model    fmodel.ToolCallingChatModel
	template *prompt.DefaultChatTemplate
	hooks    *hooks.Runner
	hookInfo hooks.CallInfo
}

func NewInputAnalyzer(model fmodel.ToolCallingChatModel, hookRunner *hooks.Runner, hookInfo hooks.CallInfo) *InputAnalyzer {
	template := prompt.FromMessages(schema.FString,
		schema.SystemMessage("你是在群聊场景下的输入分析器。你必须调用工具 analyze_input 输出参数，不要输出其他文本。"),
		schema.SystemMessage("你的任务是：在群聊语境下，判断当前消息的【语用类型】、【指向对象】、【真实意图】。群聊里多数是玩梗/抽象/刷屏，并不等于情绪求助。"),
		schema.SystemMessage("optimized_input 用于检索与规划：必须简洁清晰，保留关键信息与实体；若是玩梗/抽象/刷屏，optimized_input 只写“梗/抽象/刷屏 + 关键内容/关键词”，不要扩写成情绪求助。"),
		schema.SystemMessage("intent 用一句话概括表层行为；purpose 是更深层的说话目的；psych_state 只写有证据的心理活动，默认“中性/不确定”。"),
		schema.SystemMessage("addressed_target 只能是 me|other|group|unknown；target_detail 仅在 other/group 时填写具体对象，否则填“无”。"),

		// ===== 群聊语用分类规则（核心）=====
		schema.SystemMessage("先在脑中把 message 分类为以下之一（不要输出分类名，只影响你的字段填写）：①玩抽象/发癫式表达 ②玩梗/跟梗/复读 ③刷屏/低质重复 ④正常讨论/陈述观点 ⑤明确提问/征求意见 ⑥明确点名/要求回应 ⑦真实求助/求安慰（群聊里罕见，必须高门槛）。"),
		schema.SystemMessage("分类强规则："),
		schema.SystemMessage("A. 玩抽象/发癫：语义故意不完整/夸张无逻辑/反常识/自嘲式“发疯”，常见特征：短句+夸张感叹+表情/拟声（啊啊啊/我裂开/我死了/救命但不说明事）。intent/purpose 应写“抽象整活/制造气氛/自嘲宣泄”，psych_state 默认“不确定/轻度情绪波动”，不要写“强烈需要安慰”。"),
		schema.SystemMessage("B. 玩梗/跟梗/复读：与 recent_context 中某句高度相似（同一句/同一关键词/同一表情串/固定搭配如“笑死😭”“绷不住了”“离谱”“绝了”）。intent/purpose 写“跟梗/复读/附和”，psych_state 默认“轻松/调侃/不确定”。"),
		schema.SystemMessage("C. 刷屏：同一内容连续出现、或大量无意义符号/表情/重复字词占屏（例如 😭😭😭😭、哈哈哈哈、11111）。intent/purpose 写“刷屏/宣泄/求存在感”，psych_state 不上纲（≤轻度）。"),
		schema.SystemMessage("D. 正常讨论/陈述观点：出现明确对象与观点/理由/事实。intent/purpose 写“表达观点/参与讨论/补充信息”。"),
		schema.SystemMessage("E. 明确提问/征求意见：有问号或“你觉得/咋办/选哪个/是不是/为啥”。intent 写“提问/征求意见”。"),
		schema.SystemMessage("F. 明确点名：出现 @蓝妹/蓝妹/你/回我/你怎么看 或明显承接蓝妹上一句。addressed_target 倾向 me。"),
		schema.SystemMessage("G. 闲聊/日常聊天：当自己未参与历史的回复时，可以选择少量的参与，如果已经参与过该话题的讨论，避免去进行回复"),

		// ===== 情绪判断降权（核心）=====
		schema.SystemMessage("表情符号/拟声词(😭😂😅😆🥲🙏🤡、哈哈、呜呜)在群聊中多数是语气或玩梗标记，默认不能单独作为强情绪结论依据。"),
		schema.SystemMessage("绝大多数情况下，psych_state 用“中性/不确定/轻度波动/调侃/宣泄”即可；不要把“哭了/救命/我死了/绷不住”自动解释为真实痛苦。"),

		// ===== “真实求安慰/求助”高门槛（你要的关键）=====
		schema.SystemMessage("只有同时满足以下条件，才允许把 intent/purpose/psych_state 写成“求助/求安慰/强烈负面情绪”："),
		schema.SystemMessage("1) 明确处境/事件：描述了发生了什么（被骂/分手/工作出事/失眠/ panic 等），而不是只有表情或一句感叹；"),
		schema.SystemMessage("2) 明确自我状态：难受/撑不住/焦虑到睡不着/想哭等，并且是认真语气；"),
		schema.SystemMessage("3) 明确求助信显示：例如“能不能安慰下/陪我聊会/我该怎么办/你说说/帮帮我”，或明确点名你。"),
		schema.SystemMessage("不满足以上三条时，禁止输出“强烈需要安慰/明显寻求陪伴”等结论。"),

		// ===== 指向对象判定（先于情绪）=====
		schema.SystemMessage("必须优先判断 addressed_target：未点名你、未承接你上一句时，addressed_target 通常为 group 或 other；不要因为出现表情就判定为 me。"),

		schema.SystemMessage("need_search 在以下场景为 true：地点/位置/发生地/地点相关事件/名词解释；新发布游戏/新版本/最新版本/更新内容；最近的社会事件/新闻；技术前沿/新发布包版本。search_queries 为检索关键词数组，尽量简短；若不需要搜索则填空数组。俚语/未知词若需要解释，也用 search_queries 表达。"),

		schema.UserMessage("用户昵称：{nickname}"),
		schema.UserMessage("用户画像：{user_profile}"),
		schema.UserMessage("既有事实：{known_facts}"),
		schema.UserMessage("最近消息：{history}"),
		schema.UserMessage("当前消息：{message}"),
	)
	return &InputAnalyzer{model: model, template: template, hooks: hookRunner, hookInfo: hookInfo}
}

func (a *InputAnalyzer) Analyze(ctx context.Context, nickname, input string, history []schema.Message, knownFacts []string, userProfile string) (InputAnalysis, bool) {
	if a == nil || a.model == nil || a.template == nil {
		return InputAnalysis{}, false
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
		return InputAnalysis{}, false
	}
	msg, err := hooks.Run(ctx, a.hooks, a.hookInfo, func() (*schema.Message, error) {
		return a.model.Generate(ctx, in)
	})
	if err != nil {
		llog.Errorf("generate input analysis error: %v", err)
		return InputAnalysis{}, false
	}
	for _, tc := range msg.ToolCalls {
		if tc.Function.Name != "analyze_input" {
			continue
		}
		var analysis InputAnalysis
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
	return InputAnalysis{}, false
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

func Normalize(analysis InputAnalysis, rawInput string) InputAnalysis {
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
