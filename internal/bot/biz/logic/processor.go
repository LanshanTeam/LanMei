package logic

import (
	"LanMei/internal/bot/biz/command"
	"LanMei/internal/bot/biz/llmchat"
	"LanMei/internal/bot/biz/logic/default_plugins"
	"LanMei/internal/bot/biz/logic/process_context"
	"LanMei/internal/bot/utils/limiter"
	"LanMei/internal/bot/utils/llog"
	"LanMei/internal/bot/utils/sensitive"
	"fmt"
	"strconv"
	"strings"

	"github.com/bytedance/sonic"
	zero "github.com/wdvxdr1123/ZeroBot"
	"github.com/wdvxdr1123/ZeroBot/message"
)

type ProcessorImpl struct {
	limiter    *limiter.Limiter
	Plugins    []Plugin
	chatEngine *llmchat.ChatEngine
	Context    *process_context.Context
}

type Plugin interface {
	Name() string
	Version() string
	Description() string
	Author() string
	Enabled() bool
	Trigger(input string, ctx *zero.Ctx) bool
	Execute(input string, ctx *zero.Ctx) error
	Initialize() error
}

func defaultPlugins() []Plugin {
	return []Plugin{
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
		&default_plugins.GitHubCardPlugin{},
		&default_plugins.MusicCardPlugin{},
	}
}

func NewProcessor() *ProcessorImpl {
	processor := &ProcessorImpl{}
	if err := processor.Initialize(); err != nil {
		llog.Error("初始化Processor失败")
	}
	for _, plugin := range processor.Plugins {
		if plugin.Initialize() != nil {
			llog.Error("初始化插件 %s 失败", plugin.Name())
		}
	}

	return processor
}

func (p *ProcessorImpl) Initialize() error {
	p.limiter = limiter.NewLimiter()
	p.chatEngine = llmchat.NewChatEngine()
	p.Plugins = defaultPlugins()
	p.Context = process_context.NewContext()
	return nil
}

func (p *ProcessorImpl) RegisterPlugin(plugin ...Plugin) {
	if plugin == nil {
		llog.Error("插件不能为空")
		return
	}
	p.Plugins = append(p.Plugins, plugin...)
	for _, p := range plugin {
		if err := p.Initialize(); err != nil {
			llog.Error("初始化插件 %s 失败", p.Name())
		}
	}
}

// 处理消息
func (p *ProcessorImpl) ProcessMessage(input string, ctx *zero.Ctx) error {
	llog.Info("@事件触发！")
	p.Context.Append(ctx.Event.GroupID, process_context.Message{
		Id:       ctx.Event.MessageID.(int64),
		SenderId: ctx.Event.Sender.ID,
		Content:  input,
		AppearIn: ctx.Event.RawEvent.Time(),
	})
	msg := p.MessageProcess(input, ctx)
	if msg == "" {
		return nil
	}
	if p.Context.Behind(ctx.Event.GroupID, ctx.Event.MessageID.(int64), 2) {
		llog.Info("回复模式")
		ctx.Send(message.ReplyWithMessage(
			ctx.Event.MessageID, message.Text(msg),
		))
		return nil
	}
	ctx.Send(message.Message{
		message.Text(msg),
	})
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

func (p *ProcessorImpl) Shutdown() {
	if p == nil {
		return
	}
	if p.chatEngine != nil {
		p.chatEngine.Shutdown()
	}
}

// MessageProcess1 生成回复消息。
func (p *ProcessorImpl) MessageProcess1(input string, ctx *zero.Ctx) string {
	var msg string
	userID := fmt.Sprintf("%d", ctx.Event.UserID)
	messageID := fmt.Sprintf("%d", ctx.Event.MessageID)
	groupID := fmt.Sprintf("%d", ctx.Event.GroupID)
	if p.limiter.Deduper.Check(messageID) {
		llog.Info("重复消息: ", input)
		return ""
	} else if sensitive.HaveSensitive(input) {
		msg = ""
	} else {
		switch {
		case len(strings.TrimSpace(input)) == 0 || len(input) > 2000:
			msg = ""
		default:
			command.StaticWords(input, groupID)

			if ctx.Event.IsToMe {
				input = "蓝妹，" + input
			} else {
				for _, seg := range ctx.Event.Message {
					if seg.Type == "at" {
						qqId, _ := strconv.Atoi(seg.Data["qq"])
						json := ctx.GetStrangerInfo(int64(qqId), false)
						res := make(map[string]interface{})
						err := sonic.UnmarshalString(json.Raw, &res)
						if err != nil {
							llog.Info("反序列化错误", err)
						}
						input = res["nickname"].(string) + "，" + input
					}
				}
			}
			llog.Info("input:", input)
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
