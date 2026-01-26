package process_context

import (
	"LanMei/internal/bot/utils/llog"
	"sync"
	"time"
)

// 处理器的上下文
type Context struct {
	LatestMessageWindows sync.Map
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
	}
}

// 追加消息
func (c *Context) Append(groupId int64, msg Message) {
	if _, ok := c.LatestMessageWindows.Load(groupId); !ok {
		c.LatestMessageWindows.Store(groupId, make([]Message, 0, 30))
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
	if _, ok := c.LatestMessageWindows.Load(groupId); !ok {
		return false
	}
	window, ok := c.LatestMessageWindows.Load(groupId)
	if !ok {
		llog.Info("窗口不存在")
		return false
	}
	window1 := window.([]Message)
	for i := len(window1) - 1; i > 0; i-- {
		if window1[i].Id == messageId {
			return len(window1)-1 > i+count
		}
	}
	return true
}
