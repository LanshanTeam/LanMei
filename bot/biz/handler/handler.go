package handler

import (
	"LanMei/bot/biz/logic"
	"strings"

	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/dto/message"
	"github.com/tencent-connect/botgo/event"
)

// GroupATMessageEventHandler 实现处理 at 消息的回调
func GroupATMessageEventHandler() event.GroupATMessageEventHandler {
	return func(event *dto.WSPayload, data *dto.WSGroupATMessageData) error {
		input := strings.ToLower(message.ETLInput(data.Content))
		return logic.Processor.ProcessGroupMessage(input, data)
	}
}
