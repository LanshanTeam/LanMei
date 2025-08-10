package command

import (
	"LanMei/bot/biz/dao"
	"LanMei/bot/utils/file"
	"LanMei/bot/utils/rust_func"
	"context"
	"unicode/utf8"

	"github.com/go-ego/gse"
	"github.com/go-ego/gse/hmm/pos"
)

var seg gse.Segmenter
var posSeg pos.Segmenter
var wordClass = map[string]struct{}{"v": {}, "l": {}, "n": {}, "nr": {}, "a": {}, "vd": {}, "nz": {}, "PER": {}, "f": {}, "ns": {}, "LOC": {}, "s": {}, "nt": {}, "ORG": {}, "nw": {}, "vn": {}}

func InitWordCloud() {
	err := seg.LoadDict("./data/dict/s_1.txt", "./data/dict/t_1.txt")
	if err != nil {
		panic(err)
	}
	posSeg.WithGse(seg)
}

func StaticWords(sentence string, groupId string) {
	poss := posSeg.Cut(sentence, true)
	words := make(map[string]int64)
	for _, po := range poss {
		if _, ok := wordClass[po.Pos]; !ok {
			continue
		}
		if utf8.RuneCountInString(po.Text) < 2 {
			continue
		}
		words[po.Text]++
	}
	dao.DBManager.StaticWords(context.Background(), words, groupId)
}

func WCloud(groupID string) []byte {
	src := dao.DBManager.GetWords(context.Background(), groupID)
	picBase64 := rust_func.Wcloud(src)
	url := file.UploadPicToUrl(picBase64)
	filedata := file.UploadPicToFiledata(url, groupID)
	return filedata
}
