package core

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"time"

	retry "github.com/avast/retry-go"
	chantypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	"github.com/hyperledger-labs/yui-relayer/log"
	"github.com/hyperledger-labs/yui-relayer/otelcore/semconv"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// CreateChannel sends channel creation messages every interval until a channel is created
// TODO: add max retries or something to this function
func CreateChannel(ctx context.Context, pathName string, src, dst *ProvableChain, interval time.Duration) error {
	ctx, span := tracer.Start(ctx, "CreateChannel", WithChannelPairAttributes(src, dst))
	defer span.End()
	logger := GetChannelPairLogger(src, dst)
	defer logger.TimeTrackContext(ctx, time.Now(), "CreateChannel")

	if cont, err := checkChannelCreateReady(ctx, src, dst, logger); err != nil {
		span.SetStatus(codes.Error, err.Error())
		return err
	} else if !cont {
		return nil
	}

	failures := 0
	err := runUntilComplete(ctx, interval, func() (bool, error) {
		chanSteps, err := createChannelStep(ctx, src, dst)
		if err != nil {
			logger.ErrorContext(ctx, "failed to create channel step", err)
			return false, err
		}

		if !chanSteps.Ready() {
			logger.DebugContext(ctx, "Waiting for next channel step ...")
			return false, nil
		}

		chanSteps.Send(ctx, src, dst)

		if chanSteps.Success() {
			// In the case of success, synchronize the config file from generated channel identifiers
			if err := SyncChainConfigsFromEvents(ctx, pathName, chanSteps.SrcMsgIDs, chanSteps.DstMsgIDs, src, dst); err != nil {
				return false, err
			}

			// In the case of success and this being the last transaction
			// debug logging, log created connection and break
			if chanSteps.Last {
				logger.InfoContext(ctx, "★ Channel created")
				return true, nil
			}

			// In the case of success, reset the failures counter
			failures = 0
		} else {
			// In the case of failure, increment the failures counter and exit if this is the 3rd failure
			if failures++; failures > 2 {
				err := errors.New("Channel handshake failed")
				logger.ErrorContext(ctx, err.Error(), err)
				return false, err
			}

			logger.WarnContext(ctx, "Retrying transaction...")
			if err := wait(ctx, 5*time.Second); err != nil {
				return false, err
			}
		}

		return false, nil
	})
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	return nil
}

func checkChannelCreateReady(ctx context.Context, src, dst *ProvableChain, logger *log.RelayLogger) (bool, error) {
	srcID := src.Chain.Path().ChannelID
	dstID := dst.Chain.Path().ChannelID

	if srcID == "" && dstID == "" {
		return true, nil
	}

	getState := func(pc *ProvableChain) (chantypes.State, error) {
		if pc.Chain.Path().ChannelID == "" {
			return chantypes.UNINITIALIZED, nil
		}

		latestHeight, err := pc.LatestHeight(ctx)
		if err != nil {
			return chantypes.UNINITIALIZED, err
		}
		res, err2 := pc.QueryChannel(NewQueryContext(ctx, latestHeight))
		if err2 != nil {
			return chantypes.UNINITIALIZED, err2
		}
		return res.Channel.State, nil
	}

	srcState, srcErr := getState(src)
	if srcErr != nil {
		return false, srcErr
	}

	dstState, dstErr := getState(dst)
	if dstErr != nil {
		return false, dstErr
	}

	if srcID != "" && srcState == chantypes.UNINITIALIZED {
		return false, fmt.Errorf("src channel id is given but that channel does not exist: %s", srcID)
	}
	if dstID != "" && dstState == chantypes.UNINITIALIZED {
		return false, fmt.Errorf("dst channel id is given but that channel does not exist: %s", dstID)
	}

	if srcState == chantypes.OPEN && dstState == chantypes.OPEN {
		logger.WarnContext(ctx, "channels are already created", "src_channel_id", srcID, "dst_channel_id", dstID)
		return false, nil
	}
	return true, nil
}

func createChannelStep(ctx context.Context, src, dst *ProvableChain) (*RelayMsgs, error) {
	out := NewRelayMsgs()
	if err := validatePaths(src, dst); err != nil {
		return nil, err
	}
	// First, update the light clients to the latest header and return the header
	sh, err := NewSyncHeaders(ctx, src, dst)
	if err != nil {
		return nil, err
	}

	// Query a number of things all at once
	var (
		srcUpdateHeaders, dstUpdateHeaders []Header
	)

	err = retry.Do(func() error {
		srcUpdateHeaders, dstUpdateHeaders, err = sh.SetupBothHeadersForUpdate(ctx, src, dst)
		return err
	}, rtyAtt, rtyDel, rtyErr, retry.Context(ctx), retry.OnRetry(func(n uint, err error) {
		// logRetryUpdateHeaders(src, dst, n, err)
		if err := sh.Updates(ctx, src, dst); err != nil {
			panic(err)
		}
	}))
	if err != nil {
		return nil, err
	}

	srcChan, dstChan, settled, err := querySettledChannelPair(
		sh.GetQueryContext(ctx, src.ChainID()),
		sh.GetQueryContext(ctx, dst.ChainID()),
		src,
		dst,
		true,
	)
	if err != nil {
		return nil, err
	} else if !settled {
		return out, nil
	}

	switch {
	// Handshake hasn't been started on src or dst, relay `chanOpenInit` to src
	case srcChan.Channel.State == chantypes.UNINITIALIZED && dstChan.Channel.State == chantypes.UNINITIALIZED:
		logChannelStates(ctx, src, dst, srcChan, dstChan)
		addr := mustGetAddress(src)
		out.Src = append(out.Src,
			src.Path().ChanInit(dst.Path(), addr),
		)
	// Handshake has started on dst (1 step done), relay `chanOpenTry` and `updateClient` to src
	case srcChan.Channel.State == chantypes.UNINITIALIZED && dstChan.Channel.State == chantypes.INIT:
		logChannelStates(ctx, src, dst, srcChan, dstChan)
		addr := mustGetAddress(src)
		if len(dstUpdateHeaders) > 0 {
			out.Src = append(out.Src, src.Path().UpdateClients(dstUpdateHeaders, addr)...)
		}
		out.Src = append(out.Src, src.Path().ChanTry(dst.Path(), dstChan, addr))
	// Handshake has started on src (1 step done), relay `chanOpenTry` and `updateClient` to dst
	case srcChan.Channel.State == chantypes.INIT && dstChan.Channel.State == chantypes.UNINITIALIZED:
		logChannelStates(ctx, dst, src, dstChan, srcChan)
		addr := mustGetAddress(dst)
		if len(srcUpdateHeaders) > 0 {
			out.Dst = append(out.Dst, dst.Path().UpdateClients(srcUpdateHeaders, addr)...)
		}
		out.Dst = append(out.Dst, dst.Path().ChanTry(src.Path(), srcChan, addr))

	// Handshake has started on src (2 steps done), relay `chanOpenAck` and `updateClient` to dst
	case srcChan.Channel.State == chantypes.TRYOPEN && dstChan.Channel.State == chantypes.INIT:
		logChannelStates(ctx, dst, src, dstChan, srcChan)
		addr := mustGetAddress(dst)
		if len(srcUpdateHeaders) > 0 {
			out.Dst = append(out.Dst, dst.Path().UpdateClients(srcUpdateHeaders, addr)...)
		}
		out.Dst = append(out.Dst, dst.Path().ChanAck(src.Path(), srcChan, addr))

	// Handshake has started on dst (2 steps done), relay `chanOpenAck` and `updateClient` to src
	case srcChan.Channel.State == chantypes.INIT && dstChan.Channel.State == chantypes.TRYOPEN:
		logChannelStates(ctx, src, dst, srcChan, dstChan)
		addr := mustGetAddress(src)
		if len(dstUpdateHeaders) > 0 {
			out.Src = append(out.Src, src.Path().UpdateClients(dstUpdateHeaders, addr)...)
		}
		out.Src = append(out.Src, src.Path().ChanAck(dst.Path(), dstChan, addr))

	// Handshake has confirmed on dst (3 steps done), relay `chanOpenConfirm` and `updateClient` to src
	case srcChan.Channel.State == chantypes.TRYOPEN && dstChan.Channel.State == chantypes.OPEN:
		logChannelStates(ctx, src, dst, srcChan, dstChan)
		addr := mustGetAddress(src)
		if len(dstUpdateHeaders) > 0 {
			out.Src = append(out.Src, src.Path().UpdateClients(dstUpdateHeaders, addr)...)
		}
		out.Src = append(out.Src, src.Path().ChanConfirm(dstChan, addr))
		out.Last = true

	// Handshake has confirmed on src (3 steps done), relay `chanOpenConfirm` and `updateClient` to dst
	case srcChan.Channel.State == chantypes.OPEN && dstChan.Channel.State == chantypes.TRYOPEN:
		logChannelStates(ctx, dst, src, dstChan, srcChan)
		addr := mustGetAddress(dst)
		if len(srcUpdateHeaders) > 0 {
			out.Dst = append(out.Dst, dst.Path().UpdateClients(srcUpdateHeaders, addr)...)
		}
		out.Dst = append(out.Dst, dst.Path().ChanConfirm(srcChan, addr))
		out.Last = true
	default:
		panic(fmt.Sprintf("not implemeneted error: %v <=> %v", srcChan.Channel.State.String(), dstChan.Channel.State.String()))
	}
	return out, nil
}

func logChannelStates(ctx context.Context, src, dst *ProvableChain, srcChan, dstChan *chantypes.QueryChannelResponse) {
	logger := GetChannelPairLogger(src, dst)
	logger.InfoContext(ctx,
		"channel states",
		slog.Group("src",
			slog.Uint64("proof_height", mustGetHeight(srcChan.ProofHeight)),
			slog.String("state", srcChan.Channel.State.String()),
		),
		slog.Group("dst",
			slog.Uint64("proof_height", mustGetHeight(dstChan.ProofHeight)),
			slog.String("state", dstChan.Channel.State.String()),
		))
}

func querySettledChannelPair(
	srcCtx, dstCtx QueryContext,
	src, dst interface {
		Chain
		StateProver
	},
	prove bool,
) (*chantypes.QueryChannelResponse, *chantypes.QueryChannelResponse, bool, error) {
	logger := GetChannelPairLogger(src, dst)
	logger = &log.RelayLogger{Logger: logger.With(
		"src_height", srcCtx.Height().String(),
		"dst_height", dstCtx.Height().String(),
		"prove", prove,
	)}

	srcChan, dstChan, err := QueryChannelPair(srcCtx, dstCtx, src, dst, prove)
	if err != nil {
		logger.ErrorContext(srcCtx.Context(), "failed to query channel pair at the latest finalized height", err)
		return nil, nil, false, err
	}

	var srcLatestCtx, dstLatestCtx QueryContext
	if h, err := src.LatestHeight(srcCtx.Context()); err != nil {
		logger.ErrorContext(srcCtx.Context(), "failed to get the latest height of the src chain", err)
		return nil, nil, false, err
	} else {
		srcLatestCtx = NewQueryContext(srcCtx.Context(), h)
	}
	if h, err := dst.LatestHeight(dstCtx.Context()); err != nil {
		logger.ErrorContext(dstCtx.Context(), "failed to get the latest height of the dst chain", err)
		return nil, nil, false, err
	} else {
		dstLatestCtx = NewQueryContext(dstCtx.Context(), h)
	}

	srcLatestChan, dstLatestChan, err := QueryChannelPair(srcLatestCtx, dstLatestCtx, src, dst, false)
	if err != nil {
		logger.ErrorContext(srcCtx.Context(), "failed to query channel pair at the latest height", err)
		return nil, nil, false, err
	}

	if srcChan.Channel.String() != srcLatestChan.Channel.String() {
		logger.DebugContext(srcCtx.Context(), "src channel end in transition",
			"from", srcChan.Channel.String(),
			"to", srcLatestChan.Channel.String(),
		)
		return srcChan, dstChan, false, nil
	}
	if dstChan.Channel.String() != dstLatestChan.Channel.String() {
		logger.DebugContext(dstCtx.Context(), "dst channel end in transition",
			"from", dstChan.Channel.String(),
			"to", dstLatestChan.Channel.String(),
		)
		return srcChan, dstChan, false, nil
	}
	return srcChan, dstChan, true, nil
}

func GetChannelLogger(c Chain) *log.RelayLogger {
	return log.GetLogger().
		WithChannel(
			c.ChainID(), c.Path().PortID, c.Path().ChannelID,
		).
		WithModule("core.channel")
}

func GetChannelPairLogger(src, dst Chain) *log.RelayLogger {
	return log.GetLogger().
		WithChannelPair(
			src.ChainID(), src.Path().PortID, src.Path().ChannelID,
			dst.ChainID(), dst.Path().PortID, dst.Path().ChannelID,
		).
		WithModule("core.channel")
}

func WithChannelAttributes(c Chain) trace.SpanStartOption {
	return trace.WithAttributes(
		semconv.ChainIDKey.String(c.ChainID()),
		semconv.PortIDKey.String(c.Path().PortID),
		semconv.ChannelIDKey.String(c.Path().ChannelID),
	)
}

func WithChannelPairAttributes(src, dst Chain) trace.SpanStartOption {
	return trace.WithAttributes(slices.Concat(
		semconv.AttributeGroup("src",
			semconv.ChainIDKey.String(src.ChainID()),
			semconv.PortIDKey.String(src.Path().PortID),
			semconv.ChannelIDKey.String(src.Path().ChannelID),
		),
		semconv.AttributeGroup("dst",
			semconv.ChainIDKey.String(dst.ChainID()),
			semconv.PortIDKey.String(dst.Path().PortID),
			semconv.ChannelIDKey.String(dst.Path().ChannelID),
		),
	)...)
}
