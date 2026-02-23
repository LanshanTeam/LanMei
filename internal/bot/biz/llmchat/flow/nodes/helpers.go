package nodes

import (
	"fmt"

	flowtypes "LanMei/internal/bot/biz/llmchat/flow/types"

	"github.com/cloudwego/eino/schema"
)

const (
	replyFrequencyWindow = 8
)

func formatInterventionScores(scores *flowtypes.InterventionScores) string {
	if scores == nil {
		return "无"
	}
	return fmt.Sprintf("emotional_value=%.1f; user_emotion_need=%.1f; context_fit=%.1f; addressed_to_me=%.1f; frequency_penalty=%.1f; repeat_penalty=%.1f",
		scores.EmotionalValue,
		scores.UserEmotionNeed,
		scores.ContextFit,
		scores.AddressedToMe,
		scores.FrequencyPenalty,
		scores.RepeatPenalty,
	)
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
