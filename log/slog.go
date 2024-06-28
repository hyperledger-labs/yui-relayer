package log

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/cockroachdb/errors/withstack"
)

type RelayLogger struct {
	*slog.Logger
}

var relayLogger *RelayLogger

func InitLogger(logLevel, format, output string) error {
	// level
	var slogLevel slog.Level
	if err := slogLevel.UnmarshalText([]byte(logLevel)); err != nil {
		return fmt.Errorf("failed to unmarshal level: %v", err)
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
	args = append(args, otherArgs...)
	rl.Logger.Error(msg, args...)
}

func (rl *RelayLogger) Fatal(msg string, err error, otherArgs ...any) {
	rl.Error(msg, err, otherArgs...)
	panic(msg)
}

func GetLogger() *RelayLogger {
	return relayLogger
}

func (rl *RelayLogger) With(args ...any) *RelayLogger {
	return &RelayLogger{
		Logger: rl.Logger.With(args...),
	}
}

func (rl *RelayLogger) WithChain(
	chainID string,
) *RelayLogger {
	return rl.With(
		slog.Group("chain",
			"chain_id", chainID,
		),
	)
}

func (rl *RelayLogger) WithChainPair(
	srcChainID string,
	dstChainID string,
) *RelayLogger {
	return rl.With(
		slog.Group("chain_pair",
			slog.Group("src",
				"chain_id", srcChainID,
			),
			slog.Group("dst",
				"chain_id", dstChainID,
			),
		),
	)
}

func (rl *RelayLogger) WithClientPair(
	srcChainID, srcClientID string,
	dstChainID, dstClientID string,
) *RelayLogger {
	return rl.With(
		slog.Group("client_pair",
			slog.Group("src",
				"chain_id", srcChainID,
				"client_id", srcClientID,
			),
			slog.Group("dst",
				"chain_id", dstChainID,
				"client_id", dstClientID,
			),
		),
	)
}

func (rl *RelayLogger) WithChannel(
	chainID, portID, channelID string,
) *RelayLogger {
	return rl.With(
		slog.Group("channel",
			"chain_id", chainID,
			"port_id", portID,
			"channel_id", channelID,
		),
	)
}

func (rl *RelayLogger) WithChannelPair(
	srcChainID, srcPortID, srcChannelID string,
	dstChainID, dstPortID, dstChannelID string,
) *RelayLogger {
	return rl.With(
		slog.Group("channel_pair",
			slog.Group("src",
				"chain_id", srcChainID,
				"port_id", srcPortID,
				"channel_id", srcChannelID,
			),
			slog.Group("dst",
				"chain_id", dstChainID,
				"port_id", dstPortID,
				"channel_id", dstChannelID,
			),
		),
	)
}

func (rl *RelayLogger) WithConnectionPair(
	srcChainID, srcClientID, srcConnectionID string,
	dstChainID, dstClientID, dstConnectionID string,
) *RelayLogger {
	return rl.With(
		slog.Group("connection_pair",
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
	)
}

func (rl *RelayLogger) WithModule(
	moduleName string,
) *RelayLogger {
	return rl.With("module", moduleName)
}

func (rl *RelayLogger) TimeTrack(start time.Time, name string, otherArgs ...any) {
	elapsed := time.Since(start)
	allArgs := append([]any{"name", name, "elapsed", elapsed.Nanoseconds()}, otherArgs...)
	rl.Logger.Info("time track", allArgs...)
}
