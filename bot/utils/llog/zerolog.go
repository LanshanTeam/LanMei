package llog

import (
	"fmt"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var logger zerolog.Logger

const (
	INFO = iota
	DEBUG
	ERROR
	FATAL
	WARN
)

func InitLogger() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	logDir := "./data/log/"
	fileName := time.Now().Format("2006-01-02") + ".log"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		panic("创建日志目录失败")
	}
	file, err := os.OpenFile(logDir+fileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		panic("创建日志文件失败")
	}

	// 多路输出
	logger = log.Output(zerolog.MultiLevelWriter(
		zerolog.ConsoleWriter{Out: os.Stdout},
		file,
	))
	SetLogLevel(INFO)
	Info("ZeroLog 启动")
}

func SetLogLevel(level int) {
	switch level {
	case DEBUG:
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case ERROR:
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	case FATAL:
		zerolog.SetGlobalLevel(zerolog.FatalLevel)
	case WARN:
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
}

func Println(v ...interface{}) {
	logger.Println(v...)
}

func Info(msg string, v ...interface{}) {
	logger.Info().Msg(msg + fmt.Sprint(v...))
}

func Error(msg string, v ...interface{}) {
	logger.Error().Msg(msg + fmt.Sprint(v...))
}

func Debug(msg string, v ...interface{}) {
	logger.Debug().Msg(msg + fmt.Sprint(v...))
}

func Fatal(msg string, v ...interface{}) {
	logger.Fatal().Msg(msg + fmt.Sprint(v...))
}
