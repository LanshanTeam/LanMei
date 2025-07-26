package command

import (
	"LanMei/bot/utils/daysentence"
	"fmt"
)

func DaySentence() string {
	resp := daysentence.GetDaySentence()

	return fmt.Sprintf("每日一句：%s\n出处：%s\n，作者：%v", resp.Hitokoto, resp.From, resp.FromWho)
}
