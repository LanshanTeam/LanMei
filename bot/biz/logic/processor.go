package logic

import (
	"LanMei/bot/biz/command"
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/openapi"
)

type ProcessorImpl struct {
	Api openapi.OpenAPI
}

var Processor *ProcessorImpl

// 指令
const (
	PING        = "/ping"
	RANDOM_SIGN = "/试试手气"
	NORMAL_SIGN = "/签到"
	RANK        = "/排名"
	SET_NAME    = "/设置昵称"
	TALUO       = "/抽塔罗牌"
	DAILY_LUCK  = "/今日运势"
)

func InitProcessor(api openapi.OpenAPI) {
	Processor = &ProcessorImpl{
		Api: api,
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
func (p ProcessorImpl) ProcessGroupMessage(input string, data *dto.WSGroupATMessageData) error {
	log.Println("AT mesg")
	msg := MessageProcess(input, dto.Message(*data))
	if err := p.sendGroupReply(context.Background(), data.GroupID, msg); err != nil {
		_ = p.sendGroupReply(context.Background(), data.GroupID, genErrMessage(dto.Message(*data), err))
	}
	return nil
}

// 生成回复消息。
func MessageProcess(input string, data dto.Message) *dto.MessageToCreate {
	var msg string
	var FileInfo []byte
	MsgType := dto.TextMsg
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

	case input == TALUO:
		// 抽塔罗牌
		FileInfo, msg = command.Tarot(data.Author.ID, data.GroupID)
		MsgType = dto.RichMediaMsg

	case input == DAILY_LUCK:
		// 今日运势
		msg = command.LuckyDaily(data.Author.ID)
	default:
		// TODO：接入 AI 大模型
		msg = "收到：" + input
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
func (p ProcessorImpl) sendGroupReply(ctx context.Context, groupID string, toCreate dto.APIMessage) error {
	log.Printf("EVENT ID:%v", toCreate.GetEventID())
	if _, err := p.Api.PostGroupMessage(ctx, groupID, toCreate); err != nil {
		log.Println(err)
		return err
	}
	return nil
}
