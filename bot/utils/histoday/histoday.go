package histoday

import (
	"LanMei/bot/utils/llog"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
)

const BaseURL = "https://hao.360.cn/histoday"

var hisMatch = regexp.MustCompile(`</em>\.(.*?)</dt>`)

func GetHistory() (text string) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", BaseURL, nil)
	if err != nil {
		llog.Error("创建查询历史请求失败: %v", err)
	}
	res, err := client.Do(req)
	if err != nil {
		llog.Error("发送查询历史请求失败: %v", err)
		return
	}
	resp, err := io.ReadAll(res.Body)
	if err != nil {
		llog.Error("读取查询历史响应失败: %v", err)
		return
	}
	defer res.Body.Close()

	fmt.Println(string(resp))

	his := hisMatch.FindAllStringSubmatch(string(resp), -1)
	lines := make([]string, len(his))
	for i := range his {
		lines[i] = fmt.Sprintf("%d. %s", i+1, his[i][1])
	}
	text = fmt.Sprintf("历史上的今天\n%s", strings.Join(lines, "\n"))

	return text
}
