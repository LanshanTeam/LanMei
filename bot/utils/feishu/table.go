package feishu

import (
	"LanMei/bot/config"
	"LanMei/bot/utils/llog"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/bytedance/sonic"
)

type ReplyTable []ReplyRow

type ReplyRow interface {
	Match(words string) bool
	Reply() string
}

func NewReplyTable() *ReplyTable {
	r := make(ReplyTable, 0)
	go r.RefreshReplyList()
	return &r
}

func (r *ReplyTable) Match(words string) string {
	for _, row := range *r {
		if row.Match(words) {
			return row.Reply()
		}
	}
	return ""
}

type RegexRow struct {
	matchPattern *regexp.Regexp
	reply        string
}

var _ ReplyRow = (*RegexRow)(nil)

func NewRegexRow(pattern string, reply string) (*RegexRow, error) {
	reg, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	return &RegexRow{
		matchPattern: reg,
		reply:        reply,
	}, nil
}

func (r *RegexRow) Match(words string) bool {
	return r.matchPattern.MatchString(words)
}

func (r *RegexRow) Reply() string {
	return r.reply
}

type ContainRow struct {
	containWords string
	reply        string
}

var _ ReplyRow = (*ContainRow)(nil)

func NewContainRow(words string, reply string) *ContainRow {
	return &ContainRow{
		containWords: words,
		reply:        reply,
	}
}

func (c *ContainRow) Match(words string) bool {
	return strings.Contains(words, c.containWords)
}

func (c *ContainRow) Reply() string {
	return c.reply
}

type EqualRow struct {
	equalWords string
	reply      string
}

var _ ReplyRow = (*EqualRow)(nil)

func NewEqualRow(words string, reply string) *EqualRow {
	return &EqualRow{
		equalWords: words,
		reply:      reply,
	}
}

func (m *EqualRow) Match(words string) bool {
	return m.equalWords == words
}

func (m *EqualRow) Reply() string {
	return m.reply
}

// ======== 以下为发起刷新请求的部分 =======

type ValueRange struct {
	MajorDimension string     `json:"majorDimension"`
	Revision       int        `json:"revision"`
	Values         [][]string `json:"values"` // 我们最后的表格数据
}

type Data struct {
	Revision         int        `json:"revision"`
	SpreadsheetToken string     `json:"spreadsheetToken"`
	ValueRange       ValueRange `json:"valueRange"`
}

type GetReplyListResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data Data   `json:"data"`
}

func (rt *ReplyTable) RefreshReplyList() {
	for {
		request, err := http.NewRequest(
			"GET",
			fmt.Sprintf(
				"https://open.feishu.cn/open-apis/sheets/v2/spreadsheets/%v/values/%v?valueRenderOption=ToString",
				config.K.String("feishu.SpreadSheetToken"),
				config.K.String("feishu.SheetId"),
			),
			nil,
		)
		if err != nil {
			llog.Error("更新飞书知识库失败：", err)
			continue
		}
		request.Header.Set("Content-Type", "application/json; charset=utf-8")
		request.Header.Set("Authorization", "Bearer "+GetToken())
		c := http.Client{}
		res, err := c.Do(request)
		if err != nil {
			llog.Error("更新飞书知识库失败，请求错误", err)
			continue
		}
		resp := &GetReplyListResponse{}
		d, err := io.ReadAll(res.Body)
		if err != nil {
			llog.Error("更新飞书知识库失败，读取请求返回数据错误：", err)
			continue
		}
		sonic.Unmarshal(d, resp)
		newReplyTable := make(ReplyTable, 0)
		for i, values := range resp.Data.ValueRange.Values[1:] {
			if len(values) < 3 || values[0] == "" || values[1] == "" {
				continue
			}

			switch values[2] {
			case "全字匹配":
				newReplyTable = append(newReplyTable, NewEqualRow(values[0], values[1]))
			case "包含文字":
				newReplyTable = append(newReplyTable, NewContainRow(values[0], values[1]))
			case "正则表达式":
				r, err := NewRegexRow(values[0], values[1])
				if err != nil {
					llog.Error("正则表达式错误：", r, err)
					MarkInvalidRegexRow(config.K.String("feishu.SheetId"), fmt.Sprintf("A%d", i+2), GetToken())
					continue
				}
				newReplyTable = append(newReplyTable, r)
			default:
				newReplyTable = append(newReplyTable, NewEqualRow(values[0], values[1]))
			}
		}
		*rt = newReplyTable
		time.Sleep(30 * time.Second)
	}
}

type Style struct {
	BackColor string `json:"backColor"` // 底色，这里设置为绿色
}

type AppendStyle struct {
	Range string `json:"range"`
	Style Style  `json:"style"`
}

type TableStyle struct {
	AppendStyle AppendStyle `json:"appendStyle"`
}

type StyleResp struct {
	Code int       `json:"code"`
	Data StyleData `json:"data"`
	Msg  string    `json:"msg"`
}

type StyleData struct {
	Updates Updates `json:"updates"`
}

type Updates struct {
	Revision         int    `json:"revision"`
	SpreadsheetToken string `json:"spreadsheetToken"`
	UpdatedCells     int    `json:"updatedCells"`
	UpdatedColumns   int    `json:"updatedColumns"`
	UpdatedRange     string `json:"updatedRange"`
	UpdatedRows      int    `json:"updatedRows"`
}

// 标记错误的正则表达式为绿色
func MarkInvalidRegexRow(sheetID string, ranges string, token string) {
	style := TableStyle{AppendStyle: AppendStyle{
		Range: fmt.Sprintf("%s!%s:%s", sheetID, ranges, ranges),
		Style: Style{
			BackColor: "#21d11f",
		},
	}}
	data, err := sonic.Marshal(style)
	if err != nil {
		llog.Error("标记错误的正则表达式失败：", err)
	}
	r, err := http.NewRequest(
		"PUT",
		fmt.Sprintf("https://open.feishu.cn/open-apis/sheets/v2/spreadsheets/%v/style", GetToken()),
		bytes.NewReader(data),
	)
	if err != nil {
		llog.Error("标记错误的正则表达式失败：", err)
	}
	r.Header.Set("Content-Type", "application/json; charset=utf-8")
	r.Header.Set("Authorization", "Bearer "+GetToken())
	c := http.Client{}
	res, err := c.Do(r)
	if err != nil {
		llog.Error("标记错误的正则表达式失败，请求错误：", err)
		return
	}
	resp := StyleResp{}
	d, err := io.ReadAll(res.Body)
	if err != nil {
		llog.Error("标记错误的正则表达式失败，读取返回数据错误：", err)
		return
	}
	sonic.Unmarshal(d, resp)
	if resp.Code != 0 {
		llog.Error("标记错误的正则表达式失败：", resp.Msg)
	}
}
