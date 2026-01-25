package daysentence

import (
	"LanMei/bot/utils/llog"
	"io"
	"net/http"

	"github.com/bytedance/sonic"
)

type DaySentenceResp struct {
	ID         int    `json:"id"`
	UUID       string `json:"uuid"`
	Hitokoto   string `json:"hitokoto"`
	Type       string `json:"type"`
	From       string `json:"from"`
	FromWho    string `json:"from_who"`
	Creator    string `json:"creator"`
	CreatorUID int    `json:"creator_uid"`
	Reviewer   int    `json:"reviewer"`
	CommitFrom string `json:"commit_from"`
	CreatedAt  string `json:"created_at"`
	Length     int    `json:"length"`
}

//a 动画
//b	漫画
//c	游戏
//d	文学
//e	原创
//f	来自网络
//g	其他
//h	影视
//i	诗词
//j	网易云
//k	哲学
//l	抖机灵

var BaseURL = "https://v1.hitokoto.cn?c=a&c=b&c=d&c=i&c=j"

func GetDaySentence() *DaySentenceResp {
	r, err := http.NewRequest(http.MethodGet, BaseURL, nil)
	if err != nil {
		llog.Error("创建请求失败: %v", err)
		return nil
	}
	client := &http.Client{}
	res, err := client.Do(r)
	if err != nil {
		llog.Error("获取每日一句失败: %v", err)
		return nil
	}
	defer res.Body.Close()

	bytes, err := io.ReadAll(res.Body)
	if err != nil {
		llog.Error("读取响应体失败: %v", err)
		return nil
	}

	var resp DaySentenceResp
	err = sonic.Unmarshal(bytes, &resp)
	if err != nil {
		llog.Error("解析每日一句失败: %v", err)
		return nil
	}

	return &resp
}
