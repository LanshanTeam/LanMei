package memory

import "time"

type Fact struct {
	Subject    string  `json:"subject"`
	Content    string  `json:"content"`
	Confidence float64 `json:"confidence"`
}

type FactExtraction struct {
	Sufficient bool   `json:"sufficient"`
	Facts      []Fact `json:"facts"`
}

type FactDecision struct {
	Event    string `json:"event"`
	Text     string `json:"text"`
	TargetID any    `json:"target_id"`
	Reason   string `json:"reason"`
}

type FactRecord struct {
	ID   uint64
	Text string
}

type UserKey struct {
	GroupID string
	QQID    string
}

type knownUser struct {
	key      UserKey
	Nickname string
	lastSeen time.Time
}
