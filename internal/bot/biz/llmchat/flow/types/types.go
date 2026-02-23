package types

import "github.com/cloudwego/eino/schema"

type Request struct {
	Nickname string
	Input    string
	UserID   string
	GroupID  string
	Must     bool
	// Intervention provides optional gating scores to decide whether to reply.
	Intervention *InterventionScores
}

type InterventionScores struct {
	EmotionalValue   float64 `json:"emotional_value"`
	UserEmotionNeed  float64 `json:"user_emotion_need"`
	ContextFit       float64 `json:"context_fit"`
	AddressedToMe    float64 `json:"addressed_to_me"`
	FrequencyPenalty float64 `json:"frequency_penalty"`
	RepeatPenalty    float64 `json:"repeat_penalty"`
}

type State struct {
	Request     Request
	History     []schema.Message
	UserFacts   []string
	UserProfile string
	Reply       string
	Stop        bool
	StopReason  string
}

func NewState(req Request) *State {
	return &State{Request: req}
}

func (s *State) StopWith(reason string) {
	if s == nil {
		return
	}
	if s.Stop {
		return
	}
	s.Stop = true
	s.StopReason = reason
}
