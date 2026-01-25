package main

import (
	"LanMei/bot/biz/dao"
	"LanMei/bot/config"
	"LanMei/bot/engine"
	"LanMei/bot/utils/llog"
)

func main() {
	llog.InitLogger()
	llog.SetLogLevel(llog.DEBUG)
	config.InitKoanf()
	dao.InitDBManager()
	dao.InitSnowFlakeNode()
	engine.InitBotEngine()
}
