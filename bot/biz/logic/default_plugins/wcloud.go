package default_plugins

import (
	"LanMei/bot/biz/command"
	"strconv"

	zero "github.com/wdvxdr1123/ZeroBot"
	"github.com/wdvxdr1123/ZeroBot/message"
)

const WCLOUD = "/wcloud"

type WcloudPlugin struct{}

func (p *WcloudPlugin) Name() string {
	return "WcloudPlugin"
}

func (p *WcloudPlugin) Version() string {
	return "1.0.0"
}

func (p *WcloudPlugin) Description() string {
	return "生成词云图片"
}

func (p *WcloudPlugin) Author() string {
	return "Rinai"
}

func (p *WcloudPlugin) Enabled() bool {
	return true
}

func (p *WcloudPlugin) Trigger(input string, ctx *zero.Ctx) bool {
	return input == WCLOUD
}
func (p *WcloudPlugin) Execute(input string, ctx *zero.Ctx) error {
	File := command.WCloud(strconv.Itoa(int(ctx.Event.GroupID)))
	ctx.Send(message.Message{
		message.Image(File),
	})
	return nil
}
