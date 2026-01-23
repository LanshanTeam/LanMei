package logic

import (
	"LanMei/internal/bot/biz/command"
	"LanMei/internal/bot/biz/logic/default_plugins"
	"LanMei/internal/bot/utils/limiter"
	"LanMei/internal/bot/utils/llog"
	"LanMei/internal/bot/utils/sensitive"
	"fmt"
	"strings"

	zero "github.com/wdvxdr1123/ZeroBot"
)

type ProcessorImpl struct {
	limiter    *limiter.Limiter
	Plugins    []Plugin
	chatEngine *command.ChatEngine
}

type Plugin interface {
	Name() string
	Version() string
	Description() string
	Author() string
	Enabled() bool
	Trigger(input string, ctx *zero.Ctx) bool
	Execute(input string, ctx *zero.Ctx) error
}

var Processor *ProcessorImpl

// 指令
const (
	PING        = "/ping"
	RANDOM_SIGN = "/试试手气"
	NORMAL_SIGN = "/签到"
	RANK        = "/排名"
	SET_NAME    = "/设置昵称"
	TAROT       = "/抽塔罗牌"
	DAILY_LUCK  = "/今日运势"
	WCLOUD      = "/wcloud"
	DAYSENTENCE = "/每日一句"
	HTTPCAT1    = "/猫猫"
	HTTPCAT2    = "/哈基米"
	WEATHER     = "/天气"
	BALOGO      = "/logo"
)

var DefaultPlugins = []Plugin{
	&default_plugins.WcloudPlugin{},
	&default_plugins.CatPlugin{},
	&default_plugins.TarotPlugin{},
	&default_plugins.BaLogoPlugin{},
	&default_plugins.PingPlugin{},
	&default_plugins.SignPlugin{},
	&default_plugins.RankPlugin{},
	&default_plugins.JrlpPlugin{},
	&default_plugins.DayLuckPlugin{},
	&default_plugins.DaySentencePlugin{},
}

func NewProcessor() ProcessorImpl {
	Processor = &ProcessorImpl{
		limiter:    limiter.NewLimiter(),
		chatEngine: command.NewChatEngine(),
		Plugins:    DefaultPlugins,
	}
	return *Processor
}

// 处理消息
func (p *ProcessorImpl) ProcessMessage(input string, ctx *zero.Ctx) error {
	llog.Info("@事件触发！")
	msg := p.MessageProcess(input, ctx)
	if msg == "" {
		return nil
	}
	ctx.Send(msg)
	return nil
}

func (p *ProcessorImpl) MessageProcess(input string, ctx *zero.Ctx) string {
	for _, plugin := range p.Plugins {
		if plugin.Enabled() && plugin.Trigger(input, ctx) {
			err := plugin.Execute(input, ctx)
			if err != nil {
				llog.Error("执行插件 %s 失败: %v", plugin.Name(), err)
			}
			return ""
		}
	}

	return p.MessageProcess1(input, ctx)
}

func (p *ProcessorImpl) AddPlugin(plugin Plugin) {
	p.Plugins = append(p.Plugins, plugin)
}

// MessageProcess 生成回复消息。
func (p *ProcessorImpl) MessageProcess1(input string, ctx *zero.Ctx) string {
	var msg string
	userID := fmt.Sprintf("%d", ctx.Event.UserID)
	messageID := fmt.Sprintf("%d", ctx.Event.MessageID)
	groupID := fmt.Sprintf("%d", ctx.Event.GroupID)
	if p.limiter.Deduper.Check(messageID) {
		llog.Info("重复消息: ", input)
		return ""
	} else if sensitive.HaveSensitive(input) {
		msg = "唔唔~小蓝的数据库里没有这种词哦，要不要换个萌萌的说法呀~(>ω<)"
	} else {
		switch {
		case len(strings.TrimSpace(input)) == 0 || len(input) > 2000:
			msg = ""
		default:
			command.StaticWords(input, groupID)

			msg = p.chatEngine.ChatWithLanMei(
				ctx.Event.Sender.NickName,
				input,
				userID,
				groupID,
				ctx.Event.IsToMe,
			)
		}
	}
	return msg
}
