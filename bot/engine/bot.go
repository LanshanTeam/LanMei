package engine

import (
	"LanMei/bot/biz/handler"
	"LanMei/bot/biz/logic"
	"LanMei/bot/config"
	"LanMei/bot/utils/llog"
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/tencent-connect/botgo"
	"github.com/tencent-connect/botgo/event"
	"github.com/tencent-connect/botgo/interaction/webhook"
	"github.com/tencent-connect/botgo/token"
)

const (
	host_ = "0.0.0.0"
	port_ = 8080
	path_ = "/v1"
)

func InitBotEngine() {
	credentials := &token.QQBotCredentials{
		AppID:     config.K.String("BotAPI.AppID"),
		AppSecret: config.K.String("BotAPI.AppSecret"),
	}
	tokenSource := token.NewQQBotTokenSource(credentials)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := token.StartRefreshAccessToken(ctx, tokenSource); err != nil {
		llog.Fatal("刷新 accessToken 失败：", err)
	}
	// 初始化 openapi，正式环境
	api := botgo.NewOpenAPI(credentials.AppID, tokenSource).WithTimeout(5 * time.Second).SetDebug(true)
	logic.InitProcessor(api)
	// 注册处理函数
	_ = event.RegisterHandlers(
		// 群@机器人消息事件
		handler.GroupATMessageEventHandler(),
	)
	// 这里的 handler 用于配置 webhook 的回调验证，详见 qq 机器人开发文档。
	http.HandleFunc(path_, func(writer http.ResponseWriter, request *http.Request) {
		webhook.HTTPHandler(writer, request, credentials)
	})
	if err := http.ListenAndServe(fmt.Sprintf("%s:%d", host_, port_), nil); err != nil {
		llog.Fatal("setup server fatal:", err)
	}
}
