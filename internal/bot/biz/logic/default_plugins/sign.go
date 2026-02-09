package default_plugins

import (
	"LanMei/internal/bot/biz/command"
	"strconv"
	"strings"

	zero "github.com/wdvxdr1123/ZeroBot"
	"github.com/wdvxdr1123/ZeroBot/message"
)

const (
	RANDOM_SIGN = "/试试手气"
	NORMAL_SIGN = "/签到"
	RANK        = "/排名"
	RANK2       = "/rank"
)

type SignPlugin struct{}

func (p *SignPlugin) Name() string {
	return "SignPlugin"
}

func (p *SignPlugin) Version() string {
	return "1.0.0"
}

func (p *SignPlugin) Description() string {
	return "签到插件"
}

func (p *SignPlugin) Author() string {
	return "Rinai"
}

func (p *SignPlugin) Enabled() bool {
	return true
}

func (p *SignPlugin) Trigger(input string, ctx *zero.Ctx) bool {
	return strings.HasPrefix(input, NORMAL_SIGN) || strings.HasPrefix(input, RANDOM_SIGN)
}

func (p *SignPlugin) Execute(input string, ctx *zero.Ctx) error {
	qqId := strconv.Itoa(int(ctx.Event.Sender.ID))
	var msg string
	if strings.HasPrefix(input, NORMAL_SIGN) {
		msg = command.Sign(ctx, qqId, false)
	}
	if strings.HasPrefix(input, RANDOM_SIGN) {
		msg = command.Sign(ctx, qqId, true)
	}

	ctx.Send(message.Message{
		message.At(ctx.Event.Sender.ID),
		message.Text(msg),
	})
	return nil
}

func (p *SignPlugin) Initialize() error {
	PluginInitializeLog(p)
	return nil
}

type RankPlugin struct{}

func (p *RankPlugin) Name() string {
	return "RankPlugin"
}

func (p *RankPlugin) Version() string {
	return "1.0.0"
}

func (p *RankPlugin) Description() string {
	return "签到排名插件"
}
func (p *RankPlugin) Author() string {
	return "Rinai"
}

func (p *RankPlugin) Enabled() bool {
	return true
}

func (p *RankPlugin) Trigger(input string, ctx *zero.Ctx) bool {
	return strings.HasPrefix(input, RANK) || strings.HasPrefix(input, RANK2)
}

func (p *RankPlugin) Execute(input string, ctx *zero.Ctx) error {
	msg := command.Rank()

	ctx.Send(message.Message{
		message.At(ctx.Event.Sender.ID),
		message.Text(msg),
	})
	return nil
}

func (p *RankPlugin) Initialize() error {
	PluginInitializeLog(p)
	return nil
}
