package logic

import (
	"LanMei/bot/biz/command"
	"LanMei/bot/utils/limiter"
	"LanMei/bot/utils/llog"
	"LanMei/bot/utils/sensitive"
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/openapi"
)

type ProcessorImpl struct {
	Api     openapi.OpenAPI
	limiter *limiter.Limiter
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
	INTRO       = "/部门介绍"
)

func InitProcessor(api openapi.OpenAPI) {
	Processor = &ProcessorImpl{
		Api:     api,
		limiter: limiter.NewLimiter(),
	}
}

func genErrMessage(data dto.Message, err error) *dto.MessageToCreate {
	return &dto.MessageToCreate{
		Timestamp: time.Now().UnixMilli(),
		Content:   fmt.Sprintf("处理异常:%v", err),
		MessageReference: &dto.MessageReference{
			// 引用这条消息
			MessageID:             data.ID,
			IgnoreGetMessageError: true,
		},
		MsgID: data.ID,
	}
}

// ProcessGroupMessage 回复群消息
func (p *ProcessorImpl) ProcessGroupMessage(input string, data *dto.WSGroupATMessageData) error {
	llog.Info("@事件触发！")
	msg := p.MessageProcess(input, dto.Message(*data))
	if err := p.sendGroupReply(context.Background(), data.GroupID, msg); err != nil {
		_ = p.sendGroupReply(context.Background(), data.GroupID, genErrMessage(dto.Message(*data), err))
	}
	return nil
}

// MessageProcess 生成回复消息。
func (p *ProcessorImpl) MessageProcess(input string, data dto.Message) *dto.MessageToCreate {
	var msg string
	var FileInfo []byte
	MsgType := dto.TextMsg

	if !p.limiter.Allow(data.Author.ID) {
		// 限流
		msg = "唔...你刚刚说话太快了，蓝妹没有反应过来~"
	} else if sensitive.HaveSensitive(input) {
		// 敏感词
		msg = "唔唔~小蓝的数据库里没有这种词哦，要不要换个萌萌的说法呀~(>ω<)"
	} else {
		// 先看看是不是指令。
		switch true {
		case input == PING:
			// ping 一下
			msg = command.PingCommand()

		case input == RANDOM_SIGN:
			// 试试手气
			// 最后一个参数代表是否随机。
			msg = command.Sign(data.Author.ID, true)

		case input == NORMAL_SIGN:
			// 签到
			msg = command.Sign(data.Author.ID, false)

		case input == RANK:
			// 签到的积分排名
			msg = command.Rank()

		case strings.HasPrefix(input, SET_NAME):
			// 设置昵称
			if len(input) <= len(SET_NAME) {
				msg = "请输入你要设置的昵称😠"
				break
			}
			msg = command.SetName(data.Author.ID, input[len(SET_NAME)+1:])

		case input == TAROT:
			// 抽塔罗牌
			FileInfo, msg = command.Tarot(data.Author.ID, data.GroupID)
			if FileInfo == nil {
				msg = command.FailMsg()
				break
			}
			MsgType = dto.RichMediaMsg

		case input == DAILY_LUCK:
			// 今日运势
			msg = command.LuckyDaily(data.Author.ID)

		case len(input) == 0:
			// 随机回复词条
			msg = command.NullMsg()

		case input == WCLOUD:
			FileInfo = command.WCloud(data.GroupID)
			MsgType = dto.RichMediaMsg
			msg = ""

		case strings.HasPrefix(input, INTRO):
			// 部门介绍
			msg = command.Intro(input[len(INTRO):])

		case len(input) > 1000:
			msg = "哇~ 你是不是太着急啦？慢慢说，蓝妹在这里听着呢~"
		default:
			// TODO：接入 AI 大模型
			command.StaticWords(input)
			msg = "收到：" + input
		}
	}
	// 此处返回我们生成好的消息。
	return &dto.MessageToCreate{
		MsgType:   MsgType,
		Timestamp: time.Now().UnixMilli(),
		Content:   msg,
		MessageReference: &dto.MessageReference{
			// 引用这条消息
			MessageID:             data.ID,
			IgnoreGetMessageError: true,
		},
		Media: &dto.MediaInfo{
			FileInfo: []byte(FileInfo),
		},
		MsgID: data.ID,
	}
}

// 发送回复，这里直接用的 qq 的 API 进行回复。
func (p *ProcessorImpl) sendGroupReply(ctx context.Context, groupID string, toCreate dto.APIMessage) error {
	log.Printf("EVENT ID:%v", toCreate.GetEventID())
	if _, err := p.Api.PostGroupMessage(ctx, groupID, toCreate); err != nil {
		log.Println(err)
		return err
	}
	return nil
}
