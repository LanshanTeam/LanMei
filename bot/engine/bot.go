package engine

import (
	"LanMei/bot/biz/command"
	"LanMei/bot/biz/handler"
	"LanMei/bot/biz/logic"
	"LanMei/bot/config"
	"LanMei/bot/utils/file"
	"LanMei/bot/utils/llog"
	"LanMei/bot/utils/sensitive"
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tencent-connect/botgo"
	"github.com/tencent-connect/botgo/event"
	"github.com/tencent-connect/botgo/interaction/webhook"
	"github.com/tencent-connect/botgo/token"
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
	command.InitWordCloud()
	// 注册处理函数
	_ = event.RegisterHandlers(
		// 群@机器人消息事件
		handler.GroupATMessageEventHandler(),
	)
	file.InitFileUploader(api)
	sensitive.InitFilter()
	// 这里的 handler 用于配置 webhook 的回调验证，详见 qq 机器人开发文档。
	router := gin.Default()

	router.Any("/v1", func(c *gin.Context) {
		req := c.Request
		res := c.Writer
		webhook.HTTPHandler(res, req, credentials)
	})

	router.GET("/v1/file/:filename", func(c *gin.Context) {
		file.FileStorageHandler(c.Writer, c.Request)
	})
	router.GET("/v1/tts/:filename", func(c *gin.Context) {
		file.TTSStorageHandler(c.Writer, c.Request)
	})
	router.Run(":8080")
}
