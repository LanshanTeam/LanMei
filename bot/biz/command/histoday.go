package command

import "LanMei/bot/utils/histoday"

func Histoday() string {
	text := histoday.GetHistory()
	return text
}
