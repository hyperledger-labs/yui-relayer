package logger

import (
	"log"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type ZapLogger struct {
	Zap *zap.SugaredLogger
}

var zapLogger *ZapLogger

func init() {
	logLevelEnv := os.Getenv("LOG_LEVEL")
	var logLevel zapcore.Level
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
		logLevel = zapcore.DebugLevel
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
		OutputPaths: []string{"stdout", "/tmp/zap.log"},
	}
	logger, err := cfg.Build(
		zap.AddCallerSkip(2),
	)
	if err != nil {
		log.Fatalf("CreateLogger Error: %v", err)
	}

	zapLogger = new(ZapLogger)
	zapLogger.Zap = logger.Sugar()

}

func GetLogger() *ZapLogger {
	return zapLogger
}

func (zl *ZapLogger) Errorw(
	msg string,
	err error,
	stackKey string,
) {
	zl.Zap.Errorw(
		msg,
		zap.Error(err),
		zap.StackSkip(stackKey, 3),
	)
}

func (zl *ZapLogger) ErrorwChannel(
	msg string,
	srcChainID, srcChannelID, srcPortID string,
	dstChainID, dstChannelID, dstPortID string,
	err error,
	stackKey string,
) {
	zl.Zap.Errorw(
		msg,
		"source chain id", srcChainID,
		"source channnel id", srcChannelID,
		"source port id", srcPortID,
		"destination chain id", dstChainID,
		"destination channel id", dstChannelID,
		"destination port id", dstPortID,
		zap.Error(err),
		zap.StackSkip(stackKey, 3),
	)
}

func (zl *ZapLogger) ErrorwConnection(
	msg string,
	srcChainID, srcClientID, srcConnectionID,
	dstChainID, dstClientID, dstConnectionID string,
	err error,
	stackKey string,
) {
	zl.Zap.Errorw(
		msg,
		"source chain id", srcChainID,
		"source client id", srcClientID,
		"source connection id", srcConnectionID,
		"destination chain id", dstChainID,
		"destination client id", dstClientID,
		"destination connection id", dstConnectionID,
		zap.Error(err),
		zap.StackSkip(stackKey, 3),
	)
}

func (zl *ZapLogger) Infow(
	msg string,
	info interface{},
) {
	zl.Zap.Infow(
		msg,
		"info", info,
	)
}

func (zl *ZapLogger) InfowChannel(
	msg string,
	srcChainID, srcChannelID, srcPortID string,
	dstChainID, dstChannelID, dstPortID string,
	info interface{},
) {
	zl.Zap.Infow(
		msg,
		"source chain id", srcChainID,
		"source channnel id", srcChannelID,
		"source port id", srcPortID,
		"destination chain id", dstChainID,
		"destination channel id", dstChannelID,
		"destination port id", dstPortID,
		"info", info,
	)
}

func (zl *ZapLogger) InfowConnection(
	msg string,
	srcChainID, srcClientID, srcConnectionID string,
	dstChainID, dstClientID, dstConnectionID string,
	info interface{},
) {
	zl.Zap.Infow(
		msg,
		"source chain id", srcChainID,
		"source client id", srcClientID,
		"source connection id", srcConnectionID,
		"destination chain id", dstChainID,
		"destination client id", dstClientID,
		"destination connection id", dstConnectionID,
		"info", info,
	)
}
