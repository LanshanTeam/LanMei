package command

import (
	"LanMei/bot/utils/daysentence"
	"fmt"
)

func DaySentence() string {
	resp := daysentence.GetDaySentence()
	if resp.FromWho == "" {
		resp.FromWho = "未知"
	}
	return fmt.Sprintf("\n%s\n出处：《%s》，作者：%v", resp.Hitokoto, resp.From, resp.FromWho)
}
