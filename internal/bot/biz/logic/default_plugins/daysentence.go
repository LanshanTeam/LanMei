package default_plugins

import (
	"LanMei/internal/bot/biz/command"
	"strings"

	zero "github.com/wdvxdr1123/ZeroBot"
	"github.com/wdvxdr1123/ZeroBot/message"
)

const DAILY_SENTENCE = "/每日一句"

type DaySentencePlugin struct{}

func (p *DaySentencePlugin) Name() string {
	return "每日一句插件"
}

func (p *DaySentencePlugin) Version() string {
	return "1.0.0"
}

func (p *DaySentencePlugin) Description() string {
	return "每日一句插件，查询每日一句"
}

func (p *DaySentencePlugin) Author() string {
	return "Rinai"
}

func (p *DaySentencePlugin) Enabled() bool {
	return true
}

func (p *DaySentencePlugin) Trigger(input string, ctx *zero.Ctx) bool {
	return strings.HasPrefix(input, DAILY_SENTENCE)
}

func (p *DaySentencePlugin) Execute(input string, ctx *zero.Ctx) error {
	msg := command.DaySentence()
	ctx.Send(message.Message{
		message.At(ctx.Event.Sender.ID),
		message.Text("\n" + msg),
	})
	return nil
}

func (p *DaySentencePlugin) Initialize() error {
	PluginInitializeLog(p)
	return nil
}
