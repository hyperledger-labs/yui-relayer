package log

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"runtime"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/cockroachdb/errors/withstack"
	"github.com/samber/slog-multi"
	"go.opentelemetry.io/contrib/bridges/otelslog"
)

type RelayLogger struct {
	*slog.Logger
}

var relayLogger *RelayLogger

// TODO: Use slog.DiscardHandler when we drop support for Go 1.23 or earlier
type discardHandler struct{}

func (dh discardHandler) Enabled(context.Context, slog.Level) bool  { return false }
func (dh discardHandler) Handle(context.Context, slog.Record) error { return nil }
func (dh discardHandler) WithAttrs(attrs []slog.Attr) slog.Handler  { return dh }
func (dh discardHandler) WithGroup(name string) slog.Handler        { return dh }

func InitLogger(logLevel, format, output string, enableTelemetry bool) error {
	// output
	switch output {
	case "stdout":
		return InitLoggerWithWriter(logLevel, format, os.Stdout, enableTelemetry)
	case "stderr":
		return InitLoggerWithWriter(logLevel, format, os.Stderr, enableTelemetry)
	case "null":
		return InitLoggerWithWriter(logLevel, format, io.Discard, enableTelemetry)
	default:
		return errors.New(fmt.Sprintf("invalid log output: '%s'", output))
	}
}

func InitLoggerWithWriter(logLevel, format string, writer io.Writer, enableTelemetry bool) error {
	var handlers []slog.Handler
	if enableTelemetry {
		handlers = append(handlers, otelslog.NewHandler("github.com/hyperledger-labs/yui-relayer", otelslog.WithSource(true)))
	}

	if writer != io.Discard {
		// level
		var slogLevel slog.Level
		if err := slogLevel.UnmarshalText([]byte(logLevel)); err != nil {
			return fmt.Errorf("failed to unmarshal level: %v", err)
		}
		handlerOpts := &slog.HandlerOptions{
			Level:     slogLevel,
			AddSource: true,
		}
		// format
		switch format {
		case "text":
			handlers = append(handlers, slog.NewTextHandler(writer, handlerOpts))
		case "json":
			handlers = append(handlers, slog.NewJSONHandler(writer, handlerOpts))
		default:
			return errors.New("invalid log format")
		}
	}

	var handler slog.Handler
	if len(handlers) == 0 {
		handler = discardHandler{}
	} else if len(handlers) == 1 {
		handler = handlers[0]
	} else {
		handler = slogmulti.Fanout(handlers...)
	}

	// set global logger
	relayLogger = &RelayLogger{
		slog.New(handler),
	}
	return nil
}

func (rl *RelayLogger) log(ctx context.Context, logLevel slog.Level, skipCallDepth int, msg string, args ...any) {
	if !rl.Logger.Enabled(ctx, logLevel) {
		return
	}

	var pcs [1]uintptr
	runtime.Callers(2+skipCallDepth, pcs[:]) // skip [Callers, this func, ...]

	record := slog.NewRecord(time.Now(), logLevel, msg, pcs[0])
	record.Add(args...)

	// note that official log function also ignores Handle() error
	_ = rl.Logger.Handler().Handle(ctx, record)
}

func (rl *RelayLogger) error(ctx context.Context, skipCallDepth int, msg string, err error, otherArgs ...any) {
	err = withstack.WithStackDepth(err, 1+skipCallDepth)
	var args []any
	args = append(args, "error", err)
	args = append(args, "stack", fmt.Sprintf("%+v", err))
	args = append(args, otherArgs...)

	rl.log(ctx, slog.LevelError, 1+skipCallDepth, msg, args...)
}

func (rl *RelayLogger) Error(msg string, err error, otherArgs ...any) {
	rl.error(context.Background(), 1, msg, err, otherArgs...)
}

func (rl *RelayLogger) ErrorContext(ctx context.Context, msg string, err error, otherArgs ...any) {
	rl.error(ctx, 1, msg, err, otherArgs...)
}

func (rl *RelayLogger) Fatal(msg string, err error, otherArgs ...any) {
	rl.error(context.Background(), 1, msg, err, otherArgs...)
	panic(msg)
}

func GetLogger() *RelayLogger {
	return relayLogger
}

func (rl *RelayLogger) WithChain(
	chainID string,
) *RelayLogger {
	return &RelayLogger{
		rl.Logger.With(
			"chain_id", chainID,
		),
	}
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
			"channel_id", channelID,
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
				"channel_id", srcChannelID,
			),
			slog.Group("dst",
				"chain_id", dstChainID,
				"port_id", dstPortID,
				"channel_id", dstChannelID,
			),
		),
	}
}

func (rl *RelayLogger) WithChannelPairRelative(
	myChainID, myPortID, myChannelID string,
	cpChainID, cpPortID, cpChannelID string,
) *RelayLogger {
	return &RelayLogger{
		rl.Logger.With(
			slog.Group("self",
				"chain_id", myChainID,
				"port_id", myPortID,
				"channel_id", myChannelID,
			),
			slog.Group("cp",
				"chain_id", cpChainID,
				"port_id", cpPortID,
				"channel_id", cpChannelID,
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

func (rl *RelayLogger) WithConnectionPairRelative(
	myChainID, myClientID, myConnectionID string,
	cpChainID, cpClientID, cpConnectionID string,
) *RelayLogger {
	return &RelayLogger{
		rl.Logger.With(
			slog.Group("self",
				"chain_id", myChainID,
				"client_id", myClientID,
				"connection_id", myConnectionID,
			),
			slog.Group("cp",
				"chain_id", cpChainID,
				"client_id", cpClientID,
				"connection_id", cpConnectionID,
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

func (rl *RelayLogger) TimeTrack(start time.Time, name string, otherArgs ...any) {
	elapsed := time.Since(start)
	allArgs := append([]any{"name", name, "elapsed", elapsed.Nanoseconds()}, otherArgs...)
	rl.log(context.Background(), slog.LevelInfo, 1, "time track", allArgs...)
}

func (rl *RelayLogger) TimeTrackContext(ctx context.Context, start time.Time, name string, otherArgs ...any) {
	elapsed := time.Since(start)
	allArgs := append([]any{"name", name, "elapsed", elapsed.Nanoseconds()}, otherArgs...)
	rl.log(ctx, slog.LevelInfo, 1, "time track", allArgs...)
}
