package limiter

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// 解决由于腾讯官方不断重试导致 token 大量消耗的问题。
type deduper struct {
	mu sync.Mutex

	// 一些东西
	window  time.Duration // 窗口大小，例如 10 * time.Second
	slot    time.Duration // 桶粒度，例如 1 * time.Second
	refresh bool          // 是否在命中重复时刷新窗口（把 key 迁移到当前桶）

	// 时间轮
	buckets   []map[string]struct{} // 环形桶，每个桶是一个 key 集合
	latest    map[string]int        // key -> 最近所在桶下标
	lastTick  time.Time             // 上次推进到的“对齐时间”
	bucketIdx int                   // 当前桶下标（lastTick 对应的桶）
}

func newDeduper(window, slot time.Duration, refresh bool) *deduper {
	if slot <= 0 {
		slot = time.Second
	}
	if window < slot {
		window = slot
	}
	bkts := make([]map[string]struct{}, int(window/slot)+1)
	for i := range bkts {
		bkts[i] = make(map[string]struct{})
	}
	return &deduper{
		window:    window,
		slot:      slot,
		refresh:   refresh,
		buckets:   bkts,
		latest:    make(map[string]int),
		lastTick:  time.Now().Truncate(slot),
		bucketIdx: 0,
	}
}

// 检查窗口，实现去重功能
func (d *deduper) Check(key string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.advance()

	if idx, ok := d.latest[key]; ok {
		// 去重
		if d.refresh {
			delete(d.buckets[idx], key)
			d.buckets[d.bucketIdx][key] = struct{}{}
			d.latest[key] = d.bucketIdx
		}
		return true
	}

	// 第一次出现这个消息
	d.buckets[d.bucketIdx][key] = struct{}{}
	d.latest[key] = d.bucketIdx
	return false
}

// 刷新窗口
func (d *deduper) advance() {
	now := time.Now().Truncate(d.slot)
	if !now.After(d.lastTick) {
		return
	}
	// 看看经过了多少个槽
	steps := int(now.Sub(d.lastTick) / d.slot)
	n := len(d.buckets)
	for i := 0; i < steps; i++ {
		d.bucketIdx = (d.bucketIdx + 1) % n

		// 使用当前的 bucketIdx 先要清除过期的数据
		expired := d.buckets[d.bucketIdx]
		if len(expired) > 0 {
			for k := range expired {
				// 如果过期的数据还在 latest 中。
				if d.latest[k] == d.bucketIdx {
					delete(d.latest, k)
				}
			}
			clear(expired)
		}
	}
	d.lastTick = d.lastTick.Add(time.Duration(steps) * d.slot)
}

type Limiter struct {
	Limiters sync.Map
	Deduper  *deduper
}

func NewLimiter() *Limiter {
	return &Limiter{
		Limiters: sync.Map{},
		Deduper:  newDeduper(time.Minute, time.Second, true),
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
