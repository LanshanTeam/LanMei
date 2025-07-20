package limiter

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type Limiter struct {
	Limiters sync.Map
}

func NewLimiter() *Limiter {
	return &Limiter{
		Limiters: sync.Map{},
	}
}

func (l *Limiter) getLimiter(qqId string) *rate.Limiter {
	if limiter, exists := l.Limiters.Load(qqId); exists {
		return limiter.(*rate.Limiter)
	}
	limiter := rate.NewLimiter(rate.Every(time.Second*3), 1)
	l.Limiters.Store(qqId, limiter)
	return limiter
}

func (l *Limiter) Allow(qqId string) bool {
	limiter := l.getLimiter(qqId)
	if limiter.Allow() {
		return true
	} else {
		return false
	}
}
