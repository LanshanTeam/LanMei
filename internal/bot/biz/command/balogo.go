package command

import (
	"LanMei/internal/bot/utils/ba_logo"
)

func BALOGO(left, right string) string {
	return ba_logo.GetBALOGO(left, right)
}
