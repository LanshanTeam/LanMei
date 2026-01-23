package command

import (
	"LanMei/bot/utils/ba_logo"
	"LanMei/bot/utils/file"
)

func BALOGO(left, right string) string {
	base64 := ba_logo.GetBALOGO(left, right)
	url := file.UploadPicToUrl(base64)
	return url
}
