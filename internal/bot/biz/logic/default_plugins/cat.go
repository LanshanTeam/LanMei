package default_plugins

import (
	"LanMei/internal/bot/biz/command"
	"strconv"
	"strings"

	zero "github.com/wdvxdr1123/ZeroBot"
	"github.com/wdvxdr1123/ZeroBot/message"
)

const (
	HTTPCAT1 = "/猫猫"
	HTTPCAT2 = "/哈基米"
)

type CatPlugin struct{}

func (p *CatPlugin) Name() string {
	return "CatPlugin"
}

func (p *CatPlugin) Version() string {
	return "1.0.0"
}

func (p *CatPlugin) Description() string {
	return "发送猫猫图片"
}

func (p *CatPlugin) Author() string {
	return "Rinai"
}

func (p *CatPlugin) Enabled() bool {
	return true
}

func (p *CatPlugin) Trigger(input string, ctx *zero.Ctx) bool {
	return strings.HasPrefix(input, HTTPCAT1) || strings.HasPrefix(input, HTTPCAT2)
}
func (p *CatPlugin) Execute(input string, ctx *zero.Ctx) error {
	var File string

	groupID := strconv.Itoa(int(ctx.Event.GroupID))
	switch {
	case strings.HasPrefix(input, HTTPCAT1):
		// 猫猫1
		if len(input) == len(HTTPCAT1) {
			File = command.GetHttpCat("", groupID)
		} else {
			File = command.GetHttpCat(input[len(HTTPCAT1)+1:], groupID)
		}
	case strings.HasPrefix(input, HTTPCAT2):
		// 猫猫2
		if len(input) == len(HTTPCAT2) {
			File = command.GetHttpCat("", groupID)
		} else {
			File = command.GetHttpCat(input[len(HTTPCAT2)+1:], groupID)
		}
	}
	ctx.Send(message.Message{
		message.Image(File),
	})
	return nil
}
