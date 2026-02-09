package default_plugins

import (
	"LanMei/internal/bot/biz/command"
	"strings"

	zero "github.com/wdvxdr1123/ZeroBot"
	"github.com/wdvxdr1123/ZeroBot/message"
)

const BALOGO = "/balogo"

type BaLogoPlugin struct{}

func (p *BaLogoPlugin) Name() string {
	return "BaLogoPlugin"
}

func (p *BaLogoPlugin) Version() string {
	return "1.0.0"
}

func (p *BaLogoPlugin) Description() string {
	return "生成蔚蓝档案logo图片"
}

func (p *BaLogoPlugin) Author() string {
	return "KQ"
}

func (p *BaLogoPlugin) Enabled() bool {
	return true
}

func (p *BaLogoPlugin) Trigger(input string, ctx *zero.Ctx) bool {
	return strings.HasPrefix(input, BALOGO)
}

func (p *BaLogoPlugin) Execute(input string, ctx *zero.Ctx) error {
	parts := strings.SplitN(input[len(BALOGO)+1:], " ", 2)
	var File string
	if len(parts) != 2 {
		ctx.Send(message.Message{
			message.Text("请提供左右两部分文字哦~格式：/balogo 左文字 右文字"),
		})
		return nil
	} else {
		File = command.BALOGO(parts[0], parts[1])
	}
	ctx.Send(message.Message{
		message.Image(File),
	})
	return nil
}

func (p *BaLogoPlugin) Initialize() error {
	PluginInitializeLog(p)
	return nil
}
