package log

import (
	"fmt"
	"io"
	"os"

	"github.com/cockroachdb/errors"
	"github.com/cockroachdb/errors/withstack"
	"golang.org/x/exp/slog"
)

type RelayLogger struct {
	*slog.Logger
}

var relayLogger *RelayLogger

func InitLogger(logLevel, format, output string) error {
	// level
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
		return errors.New("invalid log level")
	}
	handlerOpts := &slog.HandlerOptions{
		Level:     slogLevel,
		AddSource: true,
	}

	// output
	var writer io.Writer
	switch output {
	case "stdout":
		writer = os.Stdout
	case "stderr":
		writer = os.Stderr
	default:
		return errors.New("invalid log output")
	}

	var slogLogger *slog.Logger
	// format
	switch format {
	case "text":
		slogLogger = slog.New(slog.NewTextHandler(
			writer,
			handlerOpts,
		))
	case "json":
		slogLogger = slog.New(slog.NewJSONHandler(
			writer,
			handlerOpts,
		))
	default:
		return errors.New("invalid log format")
	}

	// set global logger
	relayLogger = &RelayLogger{
		slogLogger,
	}
	return nil
}

func (rl *RelayLogger) Error(msg string, err error, otherArgs ...any) {
	err = withstack.WithStackDepth(err, 1)
	var args []any
	args = append(args, "error", err)
	args = append(args, "stack", fmt.Sprintf("%+v", err))
	for _, otherArg := range otherArgs {
		args = append(args, otherArg)
	}
	rl.Logger.Error(msg, args...)
}

func GetLogger() *RelayLogger {
	return relayLogger
}

func (rl *RelayLogger) WithChainPair(
	srcChainID string,
	dstChainID string,
) *RelayLogger {
	return &RelayLogger{
		rl.Logger.With(
			slog.Group("src",
				"chain_id", srcChainID,
			),
			slog.Group("dst",
				"chain_id", dstChainID,
			),
		),
	}
}

func (rl *RelayLogger) WithClientPair(
	srcChainID, srcClientID string,
	dstChainID, dstClientID string,
) *RelayLogger {
	return &RelayLogger{
		rl.Logger.With(
			slog.Group("src",
				"chain_id", srcChainID,
				"client_id", srcClientID,
			),
			slog.Group("dst",
				"chain_id", dstChainID,
				"client_id", dstClientID,
			),
		),
	}
}

func (rl *RelayLogger) WithChannel(
	chainID, portID, channelID string,
) *RelayLogger {
	return &RelayLogger{
		rl.Logger.With(
			"chain_id", chainID,
			"port_id", portID,
			"channnel_id", channelID,
		),
	}
}

func (rl *RelayLogger) WithChannelPair(
	srcChainID, srcPortID, srcChannelID string,
	dstChainID, dstPortID, dstChannelID string,
) *RelayLogger {
	return &RelayLogger{
		rl.Logger.With(
			slog.Group("src",
				"chain_id", srcChainID,
				"port_id", srcPortID,
				"channnel_id", srcChannelID,
			),
			slog.Group("dst",
				"chain_id", dstChainID,
				"port_id", dstPortID,
				"channel_id", dstChannelID,
			),
		),
	}
}

func (rl *RelayLogger) WithConnectionPair(
	srcChainID, srcClientID, srcConnectionID string,
	dstChainID, dstClientID, dstConnectionID string,
) *RelayLogger {
	return &RelayLogger{
		rl.Logger.With(
			slog.Group("src",
				"chain_id", srcChainID,
				"client_id", srcClientID,
				"connection_id", srcConnectionID,
			),
			slog.Group("dst",
				"chain_id", dstChainID,
				"client_id", dstClientID,
				"connection_id", dstConnectionID,
			),
		),
	}
}

func (rl *RelayLogger) WithModule(
	moduleName string,
) *RelayLogger {
	return &RelayLogger{
		rl.Logger.With(
			"module", moduleName,
		),
	}
}
