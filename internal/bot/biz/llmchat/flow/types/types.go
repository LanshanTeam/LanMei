package types

import "github.com/cloudwego/eino/schema"

type Request struct {
	Nickname string
	Input    string
	UserID   string
	GroupID  string
	Must     bool
}

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

type Plan struct {
	Action        string  `json:"action"`
	Intent        string  `json:"intent"`
	ReplyStyle    string  `json:"reply_style"`
	NeedMemory    bool    `json:"need_memory"`
	NeedKnowledge bool    `json:"need_knowledge"`
	NeedClarify   bool    `json:"need_clarify"`
	Confidence    float64 `json:"confidence"`
}

type State struct {
	Request      Request
	History      []schema.Message
	UserFacts    []string
	UserProfile  string
	Analysis     InputAnalysis
	Plan         Plan
	MemoryBlock  string
	Knowledge    []string
	WebSearchRaw string
	WebSearch    string
	Prompt       []*schema.Message
	Reply        string
	Stop         bool
	StopReason   string
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
