package default_plugins

import (
	zero "github.com/wdvxdr1123/ZeroBot"
	"github.com/wdvxdr1123/ZeroBot/message"
)

type PingPlugin struct{}

func (p *PingPlugin) Name() string {
	return "PingPlugin"
}

func (p *PingPlugin) Version() string {
	return "1.0.0"
}

func (p *PingPlugin) Description() string {
	return "响应 Ping 命令"
}

func (p *PingPlugin) Author() string {
	return "Rinai"
}

func (p *PingPlugin) Enabled() bool {
	return true
}

func (p *PingPlugin) Trigger(input string, ctx *zero.Ctx) bool {
	return input == "/ping"
}
func (p *PingPlugin) Execute(input string, ctx *zero.Ctx) error {
	ctx.Send(message.Message{
		message.Text("Pong!"),
	})
	return nil
}
