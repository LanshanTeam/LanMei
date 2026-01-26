package llmchat

import (
	"context"
	"sync"
	"time"
)

type memoryBatch struct {
	groupID  string
	messages []MemoryMessage
}

type MemoryWorker struct {
	manager     *MemoryManager
	interval    time.Duration
	minMessages int
	maxMessages int
	signal      chan memoryBatch
	stop        chan struct{}
	done        chan struct{}
	stopOnce    sync.Once
}

func NewMemoryWorker(manager *MemoryManager, interval time.Duration, minMessages, maxMessages int) *MemoryWorker {
	if minMessages <= 0 {
		minMessages = 4
	}
	if maxMessages <= 0 {
		maxMessages = 12
	}
	if interval <= 0 {
		interval = 10 * time.Second
	}
	return &MemoryWorker{
		manager:     manager,
		interval:    interval,
		minMessages: minMessages,
		maxMessages: maxMessages,
		signal:      make(chan memoryBatch, 128),
		stop:        make(chan struct{}),
		done:        make(chan struct{}),
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
	w.stopOnce.Do(func() {
		close(w.stop)
	})
	<-w.done
}

func (w *MemoryWorker) Enqueue(groupID string, messages []MemoryMessage) {
	w.enqueue(groupID, messages, false)
}

func (w *MemoryWorker) EnqueueBlocking(groupID string, messages []MemoryMessage) {
	w.enqueue(groupID, messages, true)
}

func (w *MemoryWorker) enqueue(groupID string, messages []MemoryMessage, block bool) {
	if w == nil || len(messages) == 0 {
		return
	}
	copied := make([]MemoryMessage, len(messages))
	copy(copied, messages)
	batch := memoryBatch{groupID: groupID, messages: copied}
	if block {
		w.signal <- batch
		return
	}
	select {
	case w.signal <- batch:
	default:
	}
}

func (w *MemoryWorker) loop() {
	if w.manager == nil {
		close(w.done)
		return
	}
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	pending := make(map[string][]MemoryMessage)
	lastAttempt := make(map[string]int)
	for {
		select {
		case batch := <-w.signal:
			pending[batch.groupID] = append(pending[batch.groupID], batch.messages...)
		case <-ticker.C:
			w.processPending(pending, lastAttempt, false)
		case <-w.stop:
			w.drainSignal(pending)
			w.processPending(pending, lastAttempt, true)
			close(w.done)
			return
		}
	}
}

func (w *MemoryWorker) drainSignal(pending map[string][]MemoryMessage) {
	for {
		select {
		case batch := <-w.signal:
			pending[batch.groupID] = append(pending[batch.groupID], batch.messages...)
		default:
			return
		}
	}
}

func (w *MemoryWorker) processPending(pending map[string][]MemoryMessage, lastAttempt map[string]int, force bool) {
	if w.manager == nil || len(pending) == 0 {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	for groupID, messages := range pending {
		if !force && lastAttempt[groupID] == len(messages) {
			continue
		}
		remaining, insufficient := w.processGroup(ctx, groupID, messages, force)
		if len(remaining) == 0 {
			delete(pending, groupID)
			delete(lastAttempt, groupID)
			continue
		}
		pending[groupID] = remaining
		if insufficient && !force {
			lastAttempt[groupID] = len(remaining)
		} else if !force {
			delete(lastAttempt, groupID)
		}
	}
}

func (w *MemoryWorker) processGroup(ctx context.Context, groupID string, messages []MemoryMessage, force bool) ([]MemoryMessage, bool) {
	for len(messages) > 0 {
		if len(messages) < w.minMessages && !force {
			return messages, false
		}
		batchSize := len(messages)
		if w.maxMessages > 0 && batchSize > w.maxMessages {
			batchSize = w.maxMessages
		}
		batch := messages[:batchSize]
		extraction := w.manager.ExtractEvent(ctx, groupID, batch, force)
		if !extraction.Sufficient && !force {
			return messages, true
		}
		if extraction.Sufficient || force {
			w.manager.StoreEvent(ctx, groupID, extraction)
		}
		messages = messages[batchSize:]
	}
	return messages, false
}
