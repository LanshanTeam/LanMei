package engine

import (
	"LanMei/internal/bot/biz/command"
	"LanMei/internal/bot/biz/logic"
	"LanMei/internal/bot/config"
	"LanMei/internal/bot/utils/file"
	"LanMei/internal/bot/utils/sensitive"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	zero "github.com/wdvxdr1123/ZeroBot"
	"github.com/wdvxdr1123/ZeroBot/driver"
)

func InitBotEngine() {
	p := logic.NewProcessor()
	command.InitWordCloud()
	file.InitFileUploader(nil)
	sensitive.InitFilter()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-shutdown
		if p != nil {
			p.Shutdown()
		}
		os.Exit(0)
	}()

	// 注册处理函数
	zero.OnMessage(func(ctx *zero.Ctx) bool {
		input := ctx.Event.Message.ExtractPlainText()
		p.ProcessMessage(input, ctx)

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

	go func() {
		_ = http.ListenAndServe(":6060", nil)
	}()
	// Web服务器用于文件
	router := gin.Default()
	router.GET("/v1/file/:filename", func(c *gin.Context) {
		file.FileStorageHandler(c.Writer, c.Request)
	})
	router.Run(":8080")
}
