package nodes

import (
	"encoding/json"
	"fmt"
	"strings"

	flowtypes "LanMei/internal/bot/biz/llmchat/flow/types"
	"LanMei/internal/bot/utils/websearch"

	"github.com/cloudwego/eino/schema"
)

const (
	replyFrequencyWindow = 8
)

func formatPlan(plan flowtypes.Plan) string {
	return fmt.Sprintf("action=%s; intent=%s; style=%s; need_memory=%t; need_knowledge=%t; need_clarify=%t",
		plan.Action, plan.Intent, plan.ReplyStyle, plan.NeedMemory, plan.NeedKnowledge, plan.NeedClarify)
}

// computeReplyScore calculates the base reply score and whether it passes hard gates.
func computeReplyScore(params map[string]interface{}) (float64, bool) {
	emotionalValue := toFloat(params["emotional_value"])
	userEmotionNeed := toFloat(params["user_emotion_need"])
	contextFit := toFloat(params["context_fit"])
	addressedToMe := toFloat(params["addressed_to_me"])
	repeatPenalty := toFloat(params["repeat_penalty"])
	frequencyPenalty := toFloat(params["frequency_penalty"])

	if emotionalValue < 45.0 || contextFit < 30.0 || repeatPenalty > 0.0 || frequencyPenalty > 20.0 {
		return 0, false
	}
	if userEmotionNeed < 40.0 && addressedToMe < 30.0 {
		return 0, false
	}

	score := emotionalValue*0.3 + userEmotionNeed*0.3 + contextFit*0.3 + addressedToMe*0.1 - min(30.0, repeatPenalty+frequencyPenalty)*0.5
	return score, true
}

func toFloat(value interface{}) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case json.Number:
		f, _ := v.Float64()
		return f
	default:
		return 0
	}
}

func recentAssistantReplies(history []schema.Message, window int) int {
	if window <= 0 {
		return 0
	}
	count := 0
	for i := len(history) - 1; i >= 0 && window > 0; i-- {
		if history[i].Role == schema.Assistant {
			count++
		}
		window--
	}
	return count
}

func formatWebSearch(results []websearch.Result) string {
	if len(results) == 0 {
		return "无"
	}
	lines := make([]string, 0, len(results))
	for _, res := range results {
		line := strings.TrimSpace(res.Title)
		if line == "" {
			continue
		}
		snippet := strings.TrimSpace(res.Snippet)
		if snippet != "" {
			line = fmt.Sprintf("%s - %s", line, snippet)
		}
		if res.URL != "" {
			line = fmt.Sprintf("%s (%s)", line, res.URL)
		}
		lines = append(lines, line)
	}
	if len(lines) == 0 {
		return "无"
	}
	return strings.Join(lines, "\n")
}
