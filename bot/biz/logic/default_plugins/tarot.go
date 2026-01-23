package default_plugins

import (
	"LanMei/bot/biz/command"
	"strings"

	zero "github.com/wdvxdr1123/ZeroBot"
	"github.com/wdvxdr1123/ZeroBot/message"
)

const TAROT = "/抽塔罗牌"

type TarotPlugin struct{}

func (p *TarotPlugin) Name() string {
	return "TarotPlugin"
}

func (p *TarotPlugin) Version() string {
	return "1.0.0"
}

func (p *TarotPlugin) Description() string {
	return "抽塔罗牌"
}

func (p *TarotPlugin) Author() string {
	return "Rinai"
}

func (p *TarotPlugin) Enabled() bool {
	return true
}

func (p *TarotPlugin) Trigger(input string, ctx *zero.Ctx) bool {
	return strings.HasPrefix(input, TAROT)
}

func (p *TarotPlugin) Execute(input string, ctx *zero.Ctx) error {
	File, msg := command.Tarot()
	if File == "" {
		msg = command.FailMsg()
		ctx.Send(message.Message{
			message.Text(msg),
		})
		return nil
	}
	ctx.Send(message.Message{
		message.At(ctx.Event.UserID),
		message.Text("\n" + msg),
		message.Image(File),
	})
	return nil
}
