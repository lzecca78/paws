package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"strings"
)

var log *zap.SugaredLogger

func init() {
	Init("info")
}

func Init(level string) {
	lvl := parseLevel(level)

	encoderCfg := zapcore.EncoderConfig{
		TimeKey:        "T",
		LevelKey:       "L",
		NameKey:        "N",
		CallerKey:      zapcore.OmitKey,
		MessageKey:     "M",
		StacktraceKey:  "",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalColorLevelEncoder,
		EncodeTime:     zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05"),
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(encoderCfg),
		zapcore.Lock(os.Stdout),
		lvl,
	)

	logger := zap.New(core)
	log = logger.Sugar()
}

func parseLevel(level string) zapcore.Level {
	switch strings.ToLower(level) {
	case "debug":
		return zapcore.DebugLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

// Wrapper functions

func Debug(args ...any) { log.Debug(args...) }
func Info(args ...any)  { log.Info(args...) }
func Warn(args ...any)  { log.Warn(args...) }
func Error(args ...any) { log.Error(args...) }

func Debugf(format string, args ...any) { log.Debugf(format, args...) }
func Infof(format string, args ...any)  { log.Infof(format, args...) }
func Warnf(format string, args ...any)  { log.Warnf(format, args...) }
func Errorf(format string, args ...any) { log.Errorf(format, args...) }
