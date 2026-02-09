package process_context

import (
	"sync"
	"time"
)

// 处理器的上下文
type Context struct {
	LatestMessageWindows sync.Map
	mu                   sync.Mutex
}

const maxWindowSize = 25

type Message struct {
	Id       int64
	SenderId int64
	Content  string
	AppearIn time.Time
}

func NewContext() *Context {
	return &Context{
		LatestMessageWindows: sync.Map{},
		// 保证内部切片的并发安全
		mu: sync.Mutex{},
	}
}

// 追加消息
func (c *Context) Append(groupId int64, msg Message) {
	c.mu.Lock()
	defer c.mu.Unlock()
	windowAny, ok := c.LatestMessageWindows.Load(groupId)
	if !ok {
		windowAny = make([]Message, 0)
	}
	window := windowAny.([]Message)
	window = append(window, msg)
	for len(window) > maxWindowSize {
		window = window[1:]
	}
	c.LatestMessageWindows.Store(groupId, window)
}

// 判断该消息是否落后 count 个消息
func (c *Context) Behind(groupId int64, messageId int64, count int) bool {
	// 没必要加锁
	windowAny, ok := c.LatestMessageWindows.Load(groupId)
	if !ok {
		return false
	}
	window := windowAny.([]Message)
	for i := len(window) - 1; i >= 0; i-- {
		if window[i].Id == messageId {
			return len(window)-1 > i+count
		}
	}
	return true
}
