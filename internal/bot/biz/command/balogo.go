package command

import (
	"LanMei/internal/bot/utils/ba_logo"
	"LanMei/internal/bot/utils/file"
)

func BALOGO(left, right string) string {
	base64 := ba_logo.GetBALOGO(left, right)
	url := file.UploadPicToUrl(base64)
	return url
}
