package llmchat

import (
	"sync"
	"time"
)

const baseReplyInterval = 6 * time.Second

type FrequencyControl struct {
	talkFrequencyAdjust float64
}

func NewFrequencyControl() *FrequencyControl {
	return &FrequencyControl{talkFrequencyAdjust: 1.0}
}

func (f *FrequencyControl) Adjust() float64 {
	if f.talkFrequencyAdjust <= 0 {
		return 1.0
	}
	return f.talkFrequencyAdjust
}

func (f *FrequencyControl) SetAdjust(value float64) {
	if value < 0.1 {
		value = 0.1
	}
	if value > 5.0 {
		value = 5.0
	}
	f.talkFrequencyAdjust = value
}

type FrequencyControlManager struct {
	mu       sync.Mutex
	controls map[string]*FrequencyControl
	lastSend map[string]time.Time
}

func NewFrequencyControlManager() *FrequencyControlManager {
	return &FrequencyControlManager{
		controls: make(map[string]*FrequencyControl),
		lastSend: make(map[string]time.Time),
	}
}

func (m *FrequencyControlManager) Get(groupID string) *FrequencyControl {
	m.mu.Lock()
	defer m.mu.Unlock()
	control, ok := m.controls[groupID]
	if !ok {
		control = NewFrequencyControl()
		m.controls[groupID] = control
	}
	return control
}

func (m *FrequencyControlManager) ShouldThrottle(groupID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	control, ok := m.controls[groupID]
	if !ok {
		control = NewFrequencyControl()
		m.controls[groupID] = control
	}
	last, ok := m.lastSend[groupID]
	if !ok {
		return false
	}
	interval := time.Duration(float64(baseReplyInterval) / control.Adjust())
	return time.Since(last) < interval
}

func (m *FrequencyControlManager) MarkSent(groupID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastSend[groupID] = time.Now()
}
