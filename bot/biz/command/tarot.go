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
	msg := "\n" + file.Words[Select][SelectMsg]
	FileInfo, ok := file.FileData.Load(url)
	if !ok {
		file.Tasks <- GroupId
		for !ok {
			FileInfo, ok = file.FileData.Load(url)
		}
	}
	if FileEx, ok := file.FileExpire.Load(url); ok {
		if time.Now().Add(20 * time.Minute).After(FileEx.(time.Time)) {
			file.Tasks <- GroupId
		}
	}
	return FileInfo.([]byte), msg
}
