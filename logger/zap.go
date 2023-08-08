package logger

import (
	"log"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type ZapLogger struct {
	Zap *zap.SugaredLogger
}

var zapLogger *ZapLogger

func InitLogger(configLevel, configEncoding string, configOutputPaths []string) {
	var logLevel zapcore.Level
	switch configLevel {
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
		Encoding: configEncoding,
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
		OutputPaths: configOutputPaths,
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

func ErrorwSugaredLogger(
	sLogger *zap.SugaredLogger,
	msg string,
	err error,
	stackKey string,
) {
	sLogger.Errorw(
		msg,
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

func InfowSugaredLogger(
	sLogger *zap.SugaredLogger,
	msg string,
	info interface{},
) {
	sLogger.Infow(
		msg,
		"info", info,
	)
}

func GetChainLogger(
	logger *zap.SugaredLogger,
	srcChainID, srcPortID string,
	dstChainID, dstPortID string,
) *zap.SugaredLogger {
	return logger.With(
		"source chain id", srcChainID,
		"source port id", srcPortID,
		"destination chain id", dstChainID,
		"destination port id", dstPortID,
	)
}

func GetChannelLogger(
	logger *zap.SugaredLogger,
	srcChainID, srcChannelID, srcPortID string,
	dstChainID, dstChannelID, dstPortID string,
) *zap.SugaredLogger {
	return logger.With(
		"source chain id", srcChainID,
		"source channnel id", srcChannelID,
		"source port id", srcPortID,
		"destination chain id", dstChainID,
		"destination channel id", dstChannelID,
		"destination port id", dstPortID,
	)
}

func GetConnectionLogger(
	logger *zap.SugaredLogger,
	srcChainID, srcClientID, srcConnectionID string,
	dstChainID, dstClientID, dstConnectionID string,
) *zap.SugaredLogger {
	return logger.With(
		"source chain id", srcChainID,
		"source client id", srcClientID,
		"source connection id", srcConnectionID,
		"destination chain id", dstChainID,
		"destination client id", dstClientID,
		"destination connection id", dstConnectionID,
	)
}

func GetModuleLogger(
	logger *zap.SugaredLogger,
	moduleName string,
) *zap.SugaredLogger {
	return logger.With("module", moduleName)
}
