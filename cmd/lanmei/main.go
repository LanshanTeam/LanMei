package main

import (
	"LanMei/internal/bot/biz/dao"
	"LanMei/internal/bot/config"
	"LanMei/internal/bot/engine"
	"LanMei/internal/bot/utils/llog"
)

func main() {
	llog.InitLogger()
	llog.SetLogLevel(llog.DEBUG)
	config.InitKoanf()
	dao.InitDBManager()
	dao.InitSnowFlakeNode()
	engine.InitBotEngine()
}
