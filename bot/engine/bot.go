package engine

import (
	"LanMei/bot/biz/command"
	"LanMei/bot/biz/logic"
	"LanMei/bot/config"
	"LanMei/bot/utils/file"
	"LanMei/bot/utils/sensitive"

	"github.com/gin-gonic/gin"
	zero "github.com/wdvxdr1123/ZeroBot"
	"github.com/wdvxdr1123/ZeroBot/driver"
)

func InitBotEngine() {
	logic.InitProcessor()
	command.InitWordCloud()
	file.InitFileUploader(nil) // TODO: 适配OneBot
	sensitive.InitFilter()

	// 注册处理函数
	zero.OnMessage(func(ctx *zero.Ctx) bool {
		// llog.Info("", ctx.Event.Sender)
		// if ctx.Event.Sender.ID != 1130157066 {
		// 	return true
		// }
		input := ctx.Event.Message.ExtractPlainText()
		logic.Processor.ProcessMessage(input, ctx)

		return true
	})

	// 启动ZeroBot
	zero.Run(&zero.Config{
		Driver: []zero.Driver{
			driver.NewHTTPClient(
				config.K.String("OneBot.ListenURL"),
				config.K.String("OneBot.accessToken"),
				config.K.String("OneBot.HttpURL"),
				config.K.String("OneBot.accessToken"),
			),
		},
	})

	// Web服务器用于文件
	router := gin.Default()
	router.GET("/v1/file/:filename", func(c *gin.Context) {
		file.FileStorageHandler(c.Writer, c.Request)
	})
	router.Run(":8080")
}
