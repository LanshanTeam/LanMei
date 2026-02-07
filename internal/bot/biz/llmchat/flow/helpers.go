package flow

import (
	"encoding/json"
	"fmt"
	"strings"

	"LanMei/internal/bot/utils/websearch"

	"github.com/cloudwego/eino/schema"
)

const (
	baseReplyScoreThreshold = 55.0
	replyFrequencyWindow    = 8
	replyPenaltyMax         = 30.0
)

func formatPlan(plan Plan) string {
	return fmt.Sprintf("action=%s; intent=%s; style=%s; need_memory=%t; need_knowledge=%t; need_clarify=%t",
		plan.Action, plan.Intent, plan.ReplyStyle, plan.NeedMemory, plan.NeedKnowledge, plan.NeedClarify)
}

// computeReplyScore calculates the base reply score and whether it passes hard gates.
func computeReplyScore(params map[string]interface{}) (float64, bool) {
	emotionalValue := toFloat(params["emotional_value"])
	userEmotionNeed := toFloat(params["user_emotion_need"])
	contextFit := toFloat(params["context_fit"])
	addressedToMe := toFloat(params["addressed_to_me"])

	if emotionalValue < 45.0 || contextFit < 30.0 {
		return 0, false
	}
	if userEmotionNeed < 40.0 && addressedToMe < 30.0 {
		return 0, false
	}

	score := emotionalValue*0.55 + userEmotionNeed*0.3 + contextFit*0.1 + addressedToMe*0.05
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

func clampPenalty(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > replyPenaltyMax {
		return replyPenaltyMax
	}
	return value
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
