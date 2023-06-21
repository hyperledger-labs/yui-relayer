package logger

import (
	"log"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var zapLogger *zap.Logger

func init() {
	logLevelEnv := os.Getenv("LOG_LEVEL")
	logLevel := zapcore.DebugLevel
	switch logLevelEnv {
	case "DEBUG":
		logLevel = zapcore.DebugLevel
	case "INFO":
		logLevel = zapcore.InfoLevel
	case "WARN":
		logLevel = zapcore.WarnLevel
	case "ERROR":
		logLevel = zapcore.ErrorLevel
	case "DPANIC":
		logLevel = zapcore.DPanicLevel
	case "PANIC":
		logLevel = zapcore.PanicLevel
	case "FATAL":
		logLevel = zapcore.FatalLevel
	default:
		logLevel = zapcore.ErrorLevel
	}
	level := zap.NewAtomicLevel()
	level.SetLevel(logLevel)
	cfg := zap.Config{
		Level:    level,
		Encoding: "json",
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        "TimeStamp",
			LevelKey:       "Severity",
			NameKey:        "Name",
			CallerKey:      "Caller",
			MessageKey:     "Body",
			EncodeLevel:    zapcore.CapitalLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.StringDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		},
		OutputPaths:      []string{"stdout", "/tmp/zap.log"},
		ErrorOutputPaths: []string{"stderr", "/tmp/zap-error.log"},
	}
	logger, err := cfg.Build()
	if err != nil {
		log.Fatalf("CreateLogger Error: %v", err)
	}
	zapLogger = logger
}

func ZapLogger() *zap.Logger {
	return zapLogger
}
