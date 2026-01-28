package hooks

import (
	"LanMei/internal/bot/utils/llog"
	"context"
	"time"
)

type CallInfo struct {
	Node  string
	Model string
}

type Hook interface {
	Before(ctx context.Context, info CallInfo)
	After(ctx context.Context, info CallInfo, duration time.Duration, err error)
}

type Runner struct {
	hooks []Hook
}

func NewRunner(hooks ...Hook) *Runner {
	copied := make([]Hook, len(hooks))
	copy(copied, hooks)
	return &Runner{hooks: copied}
}

func (r *Runner) Add(h Hook) {
	if r == nil || h == nil {
		return
	}
	r.hooks = append(r.hooks, h)
}

func Run[T any](ctx context.Context, runner *Runner, info CallInfo, fn func() (T, error)) (T, error) {
	if runner == nil || len(runner.hooks) == 0 {
		return fn()
	}
	for _, hook := range runner.hooks {
		hook.Before(ctx, info)
	}
	start := time.Now()
	out, err := fn()
	duration := time.Since(start)
	for _, hook := range runner.hooks {
		hook.After(ctx, info, duration, err)
	}
	return out, err
}

type DurationLogger struct{}

func NewDurationLogger() DurationLogger {
	return DurationLogger{}
}

func (DurationLogger) Before(ctx context.Context, info CallInfo) {}

func (DurationLogger) After(ctx context.Context, info CallInfo, duration time.Duration, err error) {
	if err != nil {
		llog.Errorf("LLM调用耗时: node=%s model=%s duration=%s err=%v", info.Node, info.Model, duration, err)
		return
	}
	llog.Infof("LLM调用耗时: node=%s model=%s duration=%s", info.Node, info.Model, duration)
}
