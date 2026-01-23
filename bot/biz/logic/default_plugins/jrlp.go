package default_plugins

import (
	"LanMei/bot/biz/command"
	"LanMei/bot/utils/llog"
	"fmt"
	"strconv"
	"strings"

	zero "github.com/wdvxdr1123/ZeroBot"
	"github.com/wdvxdr1123/ZeroBot/message"
)

const JRLP = "/今日老婆"

var AvatarUrl = "https://thirdqq.qlogo.cn/g?b=qq&nk=%v&s=640"

type JrlpPlugin struct{}

func (p *JrlpPlugin) Name() string {
	return "今日老婆插件"
}

func (p *JrlpPlugin) Version() string {
	return "1.0.0"
}

func (p *JrlpPlugin) Description() string {
	return "今日老婆插件，每日抽取一位老婆"
}

func (p *JrlpPlugin) Author() string {
	return "Rinai"
}

func (p *JrlpPlugin) Enabled() bool {
	return true
}

func (p *JrlpPlugin) Trigger(input string, ctx *zero.Ctx) bool {
	return strings.HasPrefix(input, JRLP)
}

func (p *JrlpPlugin) Execute(input string, ctx *zero.Ctx) error {
	qqId := strconv.Itoa(int(ctx.Event.Sender.ID))
	lpId, msg := command.JrlpCommand(ctx, qqId)

	avatarUrl := fmt.Sprintf(AvatarUrl, lpId)

	llog.Info(avatarUrl)
	ctx.Send(message.Message{
		message.At(ctx.Event.Sender.ID),
		message.Text("\n" + msg),
		message.Image(avatarUrl),
	})
	return nil
}
