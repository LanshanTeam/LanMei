package feishu

import (
	"LanMei/bot/config"
	"LanMei/bot/utils/llog"
	"bytes"
	"io"
	"net/http"
	"time"

	"github.com/bytedance/sonic"
)

var feishuToken = &token{}

type token struct {
	token  string
	Expire time.Time
}

type refreshTokenReq struct {
	AppId     string `json:"app_id"`
	AppSecret string `json:"app_secret"`
}

type refreshTokenResp struct {
	Code              int    `json:"code"`
	Expire            int    `json:"expire"`
	Msg               string `json:"msg"`
	TenantAccessToken string `json:"tenant_access_token"`
}

func GetToken() string {
	if feishuToken.token == "" || time.Now().After(feishuToken.Expire) {
		req := refreshTokenReq{
			AppId:     config.K.String("feishu.AppID"),
			AppSecret: config.K.String("feishu.AppSecret"),
		}
		data, _ := sonic.Marshal(req)
		request, err := http.NewRequest(
			"POST",
			"https://open.feishu.cn/open-apis/auth/v3/tenant_access_token/internal",
			bytes.NewReader(data),
		)
		if err != nil {
			llog.Error("刷新飞书 Token 失败：", err)
			return err.Error()
		}
		request.Header.Set("Content-Type", "application/json; charset=utf-8")
		c := http.Client{}
		res, err := c.Do(request)
		if err != nil {
			llog.Error("刷新飞书 Token 失败，请求错误", err)
			return err.Error()
		}
		resp := &refreshTokenResp{}
		d, err := io.ReadAll(res.Body)
		if err != nil {
			llog.Error("刷新飞书 Token 失败，读取返回值错误：", err)
			return err.Error()
		}
		sonic.Unmarshal(d, resp)
		feishuToken.token = resp.TenantAccessToken
		feishuToken.Expire = time.Now().Add(time.Duration(resp.Expire)*time.Second - 1000)
	}
	return feishuToken.token
}
