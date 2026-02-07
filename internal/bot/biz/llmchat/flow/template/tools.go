package template

import "github.com/cloudwego/eino/schema"

func BuildPlanTool() *schema.ToolInfo {
	return &schema.ToolInfo{
		Name: "plan_chat",
		Desc: "根据当前消息与上下文生成对话规划参数",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"action": {
				Type:     schema.String,
				Desc:     "reply|ask_clarify|wait",
				Required: true,
			},
			"intent": {
				Type:     schema.String,
				Desc:     "简短意图（一句话概括）",
				Required: true,
			},
			"reply_style": {
				Type:     schema.String,
				Desc:     "concise|direct|gentle",
				Required: true,
			},
			"need_memory": {
				Type:     schema.Boolean,
				Desc:     "涉及是否记得/上次/以前/往事/回忆等内容时为 true",
				Required: true,
			},
			"need_knowledge": {
				Type:     schema.Boolean,
				Desc:     "涉及蓝山/学校/工作室/姓名或组织信息时为 true",
				Required: true,
			},
			"need_clarify": {
				Type:     schema.Boolean,
				Desc:     "是否需要澄清或补充信息",
				Required: true,
			},
			"confidence": {
				Type:     schema.Number,
				Desc:     "0-1 之间的置信度",
				Required: true,
			},
		}),
	}
}

func BuildJudgeTool() *schema.ToolInfo {
	return &schema.ToolInfo{
		Name: "interested_scores",
		Desc: "群聊介入评分：评估“这次是否值得插一句”。0-100 分，分值越高越值得介入。偏好：尽量参与，但同一话题不重复回复；不当情绪安慰机器人。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"emotional_value": {
				Type:     schema.Integer,
				Desc:     "本次介入的互动收益/群聊价值（融入氛围、补一句观点、推进讨论、必要时制止刷屏）。不是安慰强度；同话题重复介入应显著降低。",
				Required: true,
			},
			"user_emotion_need": {
				Type:     schema.Integer,
				Desc:     "对方需要你回应的信号强度（被点名、明确提问、明确追问）。表情/玩梗/抽象默认低；群聊中真正求安慰较少，需有明确语义证据才高。",
				Required: true,
			},
			"context_fit": {
				Type:     schema.Integer,
				Desc:     "介入时机是否合适（不打断、话题链在你这里、你未在同话题重复发言）。若对同一话题你已回过且无新信息/追问，应降到≤30。",
				Required: true,
			},
			"addressed_to_me": {
				Type:     schema.Integer,
				Desc:     "当前消息是否指向蓝妹或在邀请你接话（@蓝妹/点名/第二人称/承接你上一句）。未指向时通常较低。",
				Required: true,
			},
			"frequency_penalty": {
				Type:     schema.Integer,
				Desc:     "频次惩罚（0-40）。最近蓝妹回复过于频繁时提高；不频繁则为 0。",
				Required: true,
			},
			"repeat_penalty": {
				Type:     schema.Integer,
				Desc:     "同话题重复惩罚（0-50）。如果已在同一话题回复过，且没有新信息/新问题/点名追问，惩罚提高。",
				Required: true,
			},
		}),
	}
}
