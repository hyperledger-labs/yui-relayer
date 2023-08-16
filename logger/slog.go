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

func InitLogger(logLevel string, format string) {
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

	var slogLogger *slog.Logger
	handlerOpts := &slog.HandlerOptions{
		Level:     slogLevel,
		AddSource: true,
	}
	if format == "text" {
		slogLogger = slog.New(slog.NewTextHandler(
			os.Stdout,
			handlerOpts,
		))
	} else {
		slogLogger = slog.New(slog.NewJSONHandler(
			os.Stdout,
			handlerOpts,
		))
	}

	relayLogger = &RelayLogger{
		*slogLogger,
	}

}

func (rl *RelayLogger) ErrorWithStack(msg string, err error) {
	cError := errors.NewWithDepth(1, err.Error())
	rl.Error(msg, "error", cError, "stack", fmt.Sprintf("%+v", cError))
}

func GetLogger() *RelayLogger {
	return relayLogger
}

func (rl *RelayLogger) WithChannel(
	srcChainID, srcChannelID, srcPortID string,
	dstChainID, dstChannelID, dstPortID string,
) *RelayLogger {
	return &RelayLogger{
		*rl.With(
			"source chain id", srcChainID,
			"source channnel id", srcChannelID,
			"source port id", srcPortID,
			"destination chain id", dstChainID,
			"destination channel id", dstChannelID,
			"destination port id", dstPortID,
		),
	}
}

func (rl *RelayLogger) WithConnection(
	srcChainID, srcClientID, srcConnectionID string,
	dstChainID, dstClientID, dstConnectionID string,
) *RelayLogger {
	return &RelayLogger{
		*rl.With(
			"source chain id", srcChainID,
			"source client id", srcClientID,
			"source connection id", srcConnectionID,
			"destination chain id", dstChainID,
			"destination client id", dstClientID,
			"destination connection id", dstConnectionID,
		),
	}
}

func (rl *RelayLogger) WithModule(
	moduleName string,
) *RelayLogger {
	return &RelayLogger{
		*rl.With(
			"module", moduleName,
		),
	}
}
