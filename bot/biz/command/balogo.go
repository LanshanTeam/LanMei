package command

import "LanMei/bot/utils/ba_logo"

func BALOGO(left, right string) []byte {
	return ba_logo.GetBALOGO(left, right)
}
