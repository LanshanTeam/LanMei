package config

import (
	"LanMei/internal/bot/utils/llog"

	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
)

var K *koanf.Koanf

func InitKoanf() {
	//初始化全局配置变量，以"."作为文件分割符
	K = koanf.New(".")

	// 加载配置
	if err := K.Load(file.Provider("./manifest/config.yaml"), yaml.Parser()); err != nil {
		llog.Fatal("Load database.yaml error:" + err.Error())
	}
	llog.Info("Koanf配置加载成功")
}
