package logger

import (
	"fmt"
	"os"

	"github.com/cockroachdb/errors"
	"golang.org/x/exp/slog"
)

type RelayLogger struct {
	slog.Logger
}

var relayLogger *RelayLogger

func InitLogger(logLevel string) {
	var slogLevel slog.Level
	switch logLevel {
	case "DEBUG":
		slogLevel = slog.LevelDebug
	case "INFO":
		slogLevel = slog.LevelInfo
	case "WARN":
		slogLevel = slog.LevelWarn
	case "ERROR":
		slogLevel = slog.LevelError
	default:
		slogLevel = slog.LevelDebug
	}
	slogLogger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slogLevel}))
	relayLogger = &RelayLogger{
		*slogLogger,
	}
}

func (rl *RelayLogger) ErrorWithStack(msg string, err error, args ...interface{}) {
	cError := errors.NewWithDepth(1, err.Error())
	rl.Error(msg, "error", cError, "stack", fmt.Sprintf("%+v", cError), args)
}

func GetLogger() *RelayLogger {
	return relayLogger
}

func GetChainLogger(
	logger *RelayLogger,
	srcChainID, srcPortID string,
	dstChainID, dstPortID string,
) *RelayLogger {
	return &RelayLogger{
		*logger.With(
			"source chain id", srcChainID,
			"source port id", srcPortID,
			"destination chain id", dstChainID,
			"destination port id", dstPortID,
		),
	}
}

func GetChannelLogger(
	logger *RelayLogger,
	srcChainID, srcChannelID, srcPortID string,
	dstChainID, dstChannelID, dstPortID string,
) *RelayLogger {
	return &RelayLogger{
		*logger.With(
			"source chain id", srcChainID,
			"source channnel id", srcChannelID,
			"source port id", srcPortID,
			"destination chain id", dstChainID,
			"destination channel id", dstChannelID,
			"destination port id", dstPortID,
		),
	}
}

func GetConnectionLogger(
	logger *RelayLogger,
	srcChainID, srcClientID, srcConnectionID string,
	dstChainID, dstClientID, dstConnectionID string,
) *RelayLogger {
	return &RelayLogger{
		*logger.With(
			"source chain id", srcChainID,
			"source client id", srcClientID,
			"source connection id", srcConnectionID,
			"destination chain id", dstChainID,
			"destination client id", dstClientID,
			"destination connection id", dstConnectionID,
		),
	}
}

func GetModuleLogger(
	logger *slog.Logger,
	moduleName string,
) *slog.Logger {
	return logger.With("module", moduleName)
}
