package default_plugins

import (
	"LanMei/internal/bot/biz/command"
	"strconv"
	"strings"

	zero "github.com/wdvxdr1123/ZeroBot"
	"github.com/wdvxdr1123/ZeroBot/message"
)

const DAILY_LUCK = "/今日运势"

type DayLuckPlugin struct{}

func (p *DayLuckPlugin) Name() string {
	return "今日运势插件"
}

func (p *DayLuckPlugin) Version() string {
	return "1.0.0"
}

func (p *DayLuckPlugin) Description() string {
	return "今日运势插件，查询每日运势"
}

func (p *DayLuckPlugin) Author() string {
	return "Rinai"
}

func (p *DayLuckPlugin) Enabled() bool {
	return true
}

func (p *DayLuckPlugin) Trigger(input string, ctx *zero.Ctx) bool {
	return strings.HasPrefix(input, DAILY_LUCK)
}

func (p *DayLuckPlugin) Execute(input string, ctx *zero.Ctx) error {
	qqId := strconv.Itoa(int(ctx.Event.Sender.ID))
	msg := command.LuckyDaily(qqId)
	ctx.Send(message.Message{
		message.At(ctx.Event.Sender.ID),
		message.Text("\n" + msg),
	})
	return nil
}
