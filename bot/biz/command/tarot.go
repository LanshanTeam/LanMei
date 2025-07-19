package command

import (
	"LanMei/bot/utils/file"
	"math/rand"
	"time"
)

func Tarot(qqId string, GroupId string) ([]byte, string) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	Select := r.Int() % 22
	SelectMsg := r.Int() % 2
	url := file.Array[Select]
	msg := file.Words[Select][SelectMsg]
	FileInfo, ok := file.FileData.Load(url)
	if !ok {
		file.Tasks <- GroupId
		for !ok {
			FileInfo, ok = file.FileData.Load(url)
		}
	}
	return FileInfo.([]byte), msg
}
