package logic

import (
	"LanMei/bot/biz/command"
	"LanMei/bot/utils/limiter"
	"LanMei/bot/utils/llog"
	"LanMei/bot/utils/sensitive"
	"fmt"
	"strings"

	zero "github.com/wdvxdr1123/ZeroBot"
)

type ProcessorImpl struct {
	limiter    *limiter.Limiter
	chatEngine *command.ChatEngine
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
	//READ        = "/朗读"
	// HISTODAY    = "/历史上的今天"
	DAYSENTENCE = "/每日一句"
	HTTPCAT1    = "/猫猫"
	HTTPCAT2    = "/哈基米"
	WEATHER     = "/天气"
	BALOGO      = "/logo"
)

func InitProcessor() {
	Processor = &ProcessorImpl{
		limiter:    limiter.NewLimiter(),
		chatEngine: command.NewChatEngine(),
	}
}

// ProcessGroupMessage 回复群消息
func (p *ProcessorImpl) ProcessMessage(input string, ctx *zero.Ctx) error {
	llog.Info("@事件触发！")
	msg := p.MessageProcess(input, ctx)
	if msg == "" {
		return nil
	}
	ctx.Send(msg)
	return nil
}

// MessageProcess 生成回复消息。
func (p *ProcessorImpl) MessageProcess(input string, ctx *zero.Ctx) string {
	var msg string
	var FileInfo []byte

	userID := fmt.Sprintf("%d", ctx.Event.UserID)
	messageID := fmt.Sprintf("%d", ctx.Event.MessageID)
	groupID := fmt.Sprintf("%d", ctx.Event.GroupID)

	if !p.limiter.Allow(userID) {
		// 限流
		msg = "唔...你刚刚说话太快了，蓝妹没有反应过来~o(≧口≦)o"
	} else if p.limiter.Deduper.Check(messageID) {
		llog.Info("重复消息: ", input)
		return ""
	} else if sensitive.HaveSensitive(input) {
		msg = "唔唔~小蓝的数据库里没有这种词哦，要不要换个萌萌的说法呀~(>ω<)"
	} else {
		// 先看看是不是指令。
		switch true {
		case strings.ToLower(input) == PING:
			// ping 一下
			msg = command.PingCommand()

		case input == RANDOM_SIGN:
			// 试试手气
			// 最后一个参数代表是否随机。
			msg = command.Sign(userID, true)

		case input == NORMAL_SIGN:
			// 签到
			msg = command.Sign(userID, false)

		case input == RANK:
			// 签到的积分排名
			msg = command.Rank()

		case strings.HasPrefix(input, SET_NAME):
			// 设置昵称
			if len(input) <= len(SET_NAME) {
				msg = "请输入你要设置的昵称😠"
			} else if len(input) >= len(SET_NAME)+30 {
				msg = "名字太长啦！蓝妹记不住呢(┬┬﹏┬┬)"
			} else {
				msg = command.SetName(userID, input[len(SET_NAME)+1:])
			}
		case input == TAROT:
			// 抽塔罗牌
			FileInfo, msg = command.Tarot(userID, groupID)
			if FileInfo == nil {
				msg = command.FailMsg()
				break
			}
			// TODO: 发送图片
			msg = "图片功能待实现"

		case input == DAILY_LUCK:
			// 今日运势
			msg = command.LuckyDaily(userID)

		case len(input) == 0:
			// 随机回复词条
			msg = command.NullMsg()

		case strings.ToLower(input) == WCLOUD:
			FileInfo = command.WCloud(groupID)
			// TODO: 发送图片
			msg = "词云功能待实现"
		// case input == HISTODAY:
		// 	msg = command.Histoday()
		//case strings.HasPrefix(input, READ):
		//	FileInfo = command.Read(input[len(READ)+1:], data.ID, data.GroupID)
		//	MsgType = dto.RichMediaMsg
		//	msg = ""
		case input == DAYSENTENCE:
			// 每日一句
			msg = command.DaySentence()
			for sensitive.HaveSensitive(msg) {
				msg = command.DaySentence()
			}

		case strings.HasPrefix(input, HTTPCAT1):
			// 猫猫1
			if len(input) == len(HTTPCAT1) {
				FileInfo = command.GetHttpCat("", groupID)
			} else {
				FileInfo = command.GetHttpCat(input[len(HTTPCAT1)+1:], groupID)
			}
			msg = "图片功能待实现"

		case strings.HasPrefix(input, HTTPCAT2):
			// 猫猫2
			if len(input) == len(HTTPCAT2) {
				FileInfo = command.GetHttpCat("", groupID)
			} else {
				FileInfo = command.GetHttpCat(input[len(HTTPCAT2)+1:], groupID)
			}
			msg = "图片功能待实现"

		case strings.HasPrefix(input, WEATHER):
			// 天气
			if len(input) == len(WEATHER) {
				msg = "请指定未来小时数哦～最大支持8小时呢~(●'◡'●)"
			} else {
				msg = command.Weather(input[len(WEATHER)+1:])
				if msg == "" {
					msg = command.FailMsg()
				}
			}

		case strings.HasPrefix(input, BALOGO):
			// 生成logo
			parts := strings.SplitN(input[len(BALOGO)+1:], " ", 2)
			if len(parts) != 2 {
				msg = "请提供左右两部分文字哦~格式：/logo 左文字 右文字"
			} else {
				FileInfo = command.BALOGO(parts[0], parts[1], groupID)
				msg = "图片功能待实现"
			}

		case len(input) > 2000:
			msg = "哇~ 你是不是太着急啦？慢慢说，蓝妹在这里听着呢~(●'◡'●)"
		default:
			// TODO：接入 AI 大模型
			command.StaticWords(input, groupID)
			msg = p.chatEngine.ChatWithLanMei(input, userID)
		}
	}
	// 此处返回我们生成好的消息。
	return msg
}
