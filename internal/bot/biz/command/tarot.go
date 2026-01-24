package command

import (
	"LanMei/internal/bot/utils/file"
	"math/rand"
	"time"
)

func Tarot() (string, string) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	Select := r.Int() % len(file.Array)
	SelectMsg := r.Int() % 2
	url := file.Array[Select]
	msg := file.Words[Select][SelectMsg]
	return url, msg
}

var failMsgs = []string{
	"呜呜~今天的塔罗牌睡着了，抽不出来呢(｡•́︿•̀｡)",
	"呀~小蓝的水晶球掉地上啦，要不要再试一次？(>﹏<)",
	"小蓝害羞ing，塔罗牌暂时抽不出来…要重试吗？",
}

func FailMsg() string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	msg := failMsgs[r.Int()%len(failMsgs)]
	return msg
}
