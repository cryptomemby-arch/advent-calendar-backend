package logger

/*
Уровни логирования:
`zap.L().Debug(...)` — для технической инфы.
`zap.L().Info(...)` — для важных событий.
`zap.L().Warn(...)` — для ошибок сети, которые не критичны (retry).
`zap.L().Error(...)` — для критических ошибок.
*/

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

func InitLogger() *zap.Logger {
	fileWriter := zapcore.AddSync(&lumberjack.Logger{
		Filename:   "logs/bot.log",
		MaxSize:    100,
		MaxBackups: 5,
		MaxAge:     28,
		Compress:   true,
	})

	fileEncoder := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())

	consoleConfig := zap.NewDevelopmentEncoderConfig()
	consoleConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	consoleEncoder := zapcore.NewConsoleEncoder(consoleConfig)

	core := zapcore.NewTee(
		zapcore.NewCore(fileEncoder, fileWriter, zap.InfoLevel),
		zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), zap.DebugLevel),
	)

	logger := zap.New(core, zap.AddCaller())

	zap.ReplaceGlobals(logger)

	return logger
}
