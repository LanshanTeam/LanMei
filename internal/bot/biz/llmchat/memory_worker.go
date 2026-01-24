package llmchat

import (
	"context"
	"time"
)

type MemoryEvent struct {
	GroupID  string
	UserID   string
	Nickname string
	Input    string
	Reply    string
}

type MemoryWorker struct {
	manager   *MemoryManager
	interval  time.Duration
	batchSize int
	signal    chan MemoryEvent
	stop      chan struct{}
}

func NewMemoryWorker(manager *MemoryManager, interval time.Duration, batchSize int) *MemoryWorker {
	return &MemoryWorker{
		manager:   manager,
		interval:  interval,
		batchSize: batchSize,
		signal:    make(chan MemoryEvent, 64),
		stop:      make(chan struct{}),
	}
}

func (w *MemoryWorker) Start() {
	if w == nil {
		return
	}
	go w.loop()
}

func (w *MemoryWorker) Stop() {
	if w == nil {
		return
	}
	close(w.stop)
}

func (w *MemoryWorker) Signal(event MemoryEvent) {
	if w == nil {
		return
	}
	select {
	case w.signal <- event:
	default:
	}
}

func (w *MemoryWorker) loop() {
	if w.manager == nil {
		return
	}
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	queue := make([]MemoryEvent, 0, w.batchSize)

	for {
		select {
		case event := <-w.signal:
			queue = append(queue, event)
		case <-ticker.C:
			w.flush(queue)
			queue = queue[:0]
		case <-w.stop:
			w.flush(queue)
			return
		}
		if len(queue) >= w.batchSize {
			w.flush(queue)
			queue = queue[:0]
		}
	}
}

func (w *MemoryWorker) flush(events []MemoryEvent) {
	if len(events) == 0 || w.manager == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	grouped := make(map[string][]MemoryEvent)
	for _, event := range events {
		grouped[event.GroupID] = append(grouped[event.GroupID], event)
	}
	for groupID, groupEvents := range grouped {
		w.manager.StoreBatch(ctx, groupID, groupEvents)
	}
}
