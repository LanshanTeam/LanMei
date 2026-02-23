package reactlog

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

const (
	defaultWindowDuration = 30 * time.Minute
	defaultMaxEntries     = 60
	defaultMaxContent     = 280
)

type StepKind string

const (
	StepThought     StepKind = "thought"
	StepAction      StepKind = "action"
	StepObservation StepKind = "observation"
	StepFinal       StepKind = "final"
)

type Step struct {
	At      time.Time
	Kind    StepKind
	Content string
}

type Window struct {
	mu         sync.Mutex
	window     time.Duration
	maxEntries int
	maxContent int
	groups     map[string][]Step
}

type groupIDKey struct{}
type queryKey struct{}

func ContextWithGroupID(ctx context.Context, groupID string) context.Context {
	if groupID == "" {
		return ctx
	}
	return context.WithValue(ctx, groupIDKey{}, groupID)
}

func ContextWithQuery(ctx context.Context, query string) context.Context {
	if query == "" {
		return ctx
	}
	return context.WithValue(ctx, queryKey{}, query)
}

func GroupIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v, ok := ctx.Value(groupIDKey{}).(string); ok {
		return v
	}
	return ""
}

func QueryFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v, ok := ctx.Value(queryKey{}).(string); ok {
		return v
	}
	return ""
}

func NewWindow(window time.Duration, maxEntries, maxContent int) *Window {
	if window <= 0 {
		window = defaultWindowDuration
	}
	if maxEntries <= 0 {
		maxEntries = defaultMaxEntries
	}
	if maxContent <= 0 {
		maxContent = defaultMaxContent
	}
	return &Window{
		window:     window,
		maxEntries: maxEntries,
		maxContent: maxContent,
		groups:     make(map[string][]Step),
	}
}

func (w *Window) Append(groupID string, kind StepKind, content string) {
	if w == nil {
		return
	}
	content = strings.TrimSpace(content)
	if groupID == "" || content == "" {
		return
	}
	if w.maxContent > 0 && len([]rune(content)) > w.maxContent {
		runes := []rune(content)
		content = string(runes[:w.maxContent]) + "…"
	}
	step := Step{At: time.Now(), Kind: kind, Content: content}
	w.mu.Lock()
	defer w.mu.Unlock()
	steps := append(w.groups[groupID], step)
	steps = w.pruneLocked(steps)
	if w.maxEntries > 0 && len(steps) > w.maxEntries {
		steps = steps[len(steps)-w.maxEntries:]
	}
	w.groups[groupID] = steps
}

func (w *Window) Snapshot(groupID string) []Step {
	if w == nil || groupID == "" {
		return nil
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	steps := w.pruneLocked(w.groups[groupID])
	if len(steps) == 0 {
		w.groups[groupID] = nil
		return nil
	}
	copied := make([]Step, len(steps))
	copy(copied, steps)
	w.groups[groupID] = steps
	return copied
}

func (w *Window) Format(groupID string, maxChars int) string {
	steps := w.Snapshot(groupID)
	if len(steps) == 0 {
		return "无"
	}
	lines := make([]string, 0, len(steps))
	used := 0
	for i := len(steps) - 1; i >= 0; i-- {
		line := fmt.Sprintf("%s [%s] %s", steps[i].At.Format("15:04:05"), steps[i].Kind, steps[i].Content)
		if maxChars > 0 && used+len(line) > maxChars {
			break
		}
		lines = append(lines, line)
		used += len(line)
	}
	if len(lines) == 0 {
		return "无"
	}
	for i, j := 0, len(lines)-1; i < j; i, j = i+1, j-1 {
		lines[i], lines[j] = lines[j], lines[i]
	}
	return strings.Join(lines, "\n")
}

func (w *Window) pruneLocked(steps []Step) []Step {
	if w.window <= 0 || len(steps) == 0 {
		return steps
	}
	cutoff := time.Now().Add(-w.window)
	idx := 0
	for idx < len(steps) {
		if steps[idx].At.After(cutoff) {
			break
		}
		idx++
	}
	if idx == 0 {
		return steps
	}
	if idx >= len(steps) {
		return nil
	}
	return steps[idx:]
}
