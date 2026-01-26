package process_context

import (
	"LanMei/internal/bot/utils/llog"
	"sync"
	"time"
)

// 处理器的上下文
type Context struct {
	LatestMessageWindows sync.Map
	mu                   sync.Mutex
}

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
	if _, ok := c.LatestMessageWindows.Load(groupId); !ok {
		c.LatestMessageWindows.Store(groupId, make([]Message, 0))
	}
	window, ok := c.LatestMessageWindows.Load(groupId)
	if !ok {
		llog.Info("窗口不存在")
		return
	}
	window1 := window.([]Message)
	window1 = append(window1, msg)
	for len(window1) > 25 {
		window1 = window1[1:]
	}
	c.LatestMessageWindows.Store(groupId, window1)
}

// 判断该消息是否落后 count 个消息
func (c *Context) Behind(groupId int64, messageId int64, count int) bool {
	// 没必要加锁
	if _, ok := c.LatestMessageWindows.Load(groupId); !ok {
		return false
	}
	window, ok := c.LatestMessageWindows.Load(groupId)
	if !ok {
		llog.Info("窗口不存在")
		return false
	}
	window1 := window.([]Message)
	for i := len(window1) - 1; i >= 0; i-- {
		if window1[i].Id == messageId {
			return len(window1)-1 > i+count
		}
	}
	return true
}
