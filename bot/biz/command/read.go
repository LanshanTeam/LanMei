package command

import (
	"LanMei/bot/utils/file"
	"LanMei/bot/utils/llog"
	"LanMei/bot/utils/tts"
)

func Read(text string, id string, groupID string) []byte {
	filename := tts.TTS(text, id)
	if filename == "" {
		llog.Error("TTS 处理失败")
		return []byte("TTS 处理失败，请稍后再试")
	}

	url := file.UploadSilkToUrl(filename)
	filedata := file.UploadSilkToFiledata(url, groupID)

	return filedata
}
