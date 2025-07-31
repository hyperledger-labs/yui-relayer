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
	"golang.org/x/sync/errgroup"
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

type queryCreateChannelStateResult struct {
	updateHeaders []Header
	channel       *chantypes.QueryChannelResponse
	settled       bool
}

func queryCreateChannelState(queryCtx QueryContext, sh SyncHeaders, prover, counterparty *ProvableChain) (*queryCreateChannelStateResult, error) {
	var ret queryCreateChannelStateResult
	logger := GetChannelPairLoggerRelative(prover, counterparty)

	err := retry.Do(func() error {
		var err error
		ret.updateHeaders, err = sh.SetupHeadersForUpdate(queryCtx.Context(), prover, counterparty)
		if err != nil {
			return err
		}
		return nil
	}, rtyAtt, rtyDel, rtyErr, retry.Context(queryCtx.Context()), retry.OnRetry(func(n uint, err error) {
		// logRetryUpdateHeaders(src, dst, n, err)
		if err := sh.Updates(queryCtx.Context(), prover, counterparty); err != nil {
			panic(err)
		}
	}))
	if err != nil {
		return nil, err
	}

	ret.channel, ret.settled, err = querySettledChannel(queryCtx, logger, prover, true)
	if err != nil {
		return nil, err
	}

	return &ret, nil
}

func createChannelStep(ctx context.Context, src, dst *ProvableChain) (*RelayMsgs, error) {
	out := NewRelayMsgs()
	if err := validatePaths(src, dst); err != nil {
		return nil, err
	}

	var (
		srcState, dstState *queryCreateChannelStateResult
	)

	// First, update the light clients to the latest header and return the header
	sh, err := NewSyncHeaders(ctx, src, dst)
	if err != nil {
		return nil, err
	}

	{
		var eg = new(errgroup.Group)
		srcStream := make(chan *queryCreateChannelStateResult, 1)
		dstStream := make(chan *queryCreateChannelStateResult, 1)
		defer close(srcStream)
		defer close(dstStream)

		srcCtx := sh.GetQueryContext(ctx, src.ChainID())
		dstCtx := sh.GetQueryContext(ctx, dst.ChainID())
		eg.Go(func() error {
			state, err := queryCreateChannelState(srcCtx, sh, src, dst)
			if err != nil {
				return err
			}
			srcStream <- state
			return nil
		})
		eg.Go(func() error {
			state, err := queryCreateChannelState(dstCtx, sh, dst, src)
			if err != nil {
				return err
			}
			dstStream <- state
			return nil
		})
		var err error
		if err = eg.Wait(); err != nil {
			return nil, err
		}
		srcState = <-srcStream
		dstState = <-dstStream
	}
	if !srcState.settled || !dstState.settled {
		return out, nil
	}

	switch {
	// Handshake hasn't been started on src or dst, relay `chanOpenInit` to src
	case srcState.channel.Channel.State == chantypes.UNINITIALIZED && dstState.channel.Channel.State == chantypes.UNINITIALIZED:
		logChannelStates(ctx, src, dst, srcState.channel, dstState.channel)
		addr := mustGetAddress(src)
		out.Src = append(out.Src,
			src.Path().ChanInit(dst.Path(), addr),
		)
	// Handshake has started on dst (1 step done), relay `chanOpenTry` and `updateClient` to src
	case srcState.channel.Channel.State == chantypes.UNINITIALIZED && dstState.channel.Channel.State == chantypes.INIT:
		logChannelStates(ctx, src, dst, srcState.channel, dstState.channel)
		addr := mustGetAddress(src)
		if len(dstState.updateHeaders) > 0 {
			out.Src = append(out.Src, src.Path().UpdateClients(dstState.updateHeaders, addr)...)
		}
		out.Src = append(out.Src, src.Path().ChanTry(dst.Path(), dstState.channel, addr))
	// Handshake has started on src (1 step done), relay `chanOpenTry` and `updateClient` to dst
	case srcState.channel.Channel.State == chantypes.INIT && dstState.channel.Channel.State == chantypes.UNINITIALIZED:
		logChannelStates(ctx, dst, src, dstState.channel, srcState.channel)
		addr := mustGetAddress(dst)
		if len(srcState.updateHeaders) > 0 {
			out.Dst = append(out.Dst, dst.Path().UpdateClients(srcState.updateHeaders, addr)...)
		}
		out.Dst = append(out.Dst, dst.Path().ChanTry(src.Path(), srcState.channel, addr))

	// Handshake has started on src (2 steps done), relay `chanOpenAck` and `updateClient` to dst
	case srcState.channel.Channel.State == chantypes.TRYOPEN && dstState.channel.Channel.State == chantypes.INIT:
		logChannelStates(ctx, dst, src, dstState.channel, srcState.channel)
		addr := mustGetAddress(dst)
		if len(srcState.updateHeaders) > 0 {
			out.Dst = append(out.Dst, dst.Path().UpdateClients(srcState.updateHeaders, addr)...)
		}
		out.Dst = append(out.Dst, dst.Path().ChanAck(src.Path(), srcState.channel, addr))

	// Handshake has started on dst (2 steps done), relay `chanOpenAck` and `updateClient` to src
	case srcState.channel.Channel.State == chantypes.INIT && dstState.channel.Channel.State == chantypes.TRYOPEN:
		logChannelStates(ctx, src, dst, srcState.channel, dstState.channel)
		addr := mustGetAddress(src)
		if len(dstState.updateHeaders) > 0 {
			out.Src = append(out.Src, src.Path().UpdateClients(dstState.updateHeaders, addr)...)
		}
		out.Src = append(out.Src, src.Path().ChanAck(dst.Path(), dstState.channel, addr))

	// Handshake has confirmed on dst (3 steps done), relay `chanOpenConfirm` and `updateClient` to src
	case srcState.channel.Channel.State == chantypes.TRYOPEN && dstState.channel.Channel.State == chantypes.OPEN:
		logChannelStates(ctx, src, dst, srcState.channel, dstState.channel)
		addr := mustGetAddress(src)
		if len(dstState.updateHeaders) > 0 {
			out.Src = append(out.Src, src.Path().UpdateClients(dstState.updateHeaders, addr)...)
		}
		out.Src = append(out.Src, src.Path().ChanConfirm(dstState.channel, addr))
		out.Last = true

	// Handshake has confirmed on src (3 steps done), relay `chanOpenConfirm` and `updateClient` to dst
	case srcState.channel.Channel.State == chantypes.OPEN && dstState.channel.Channel.State == chantypes.TRYOPEN:
		logChannelStates(ctx, dst, src, dstState.channel, srcState.channel)
		addr := mustGetAddress(dst)
		if len(srcState.updateHeaders) > 0 {
			out.Dst = append(out.Dst, dst.Path().UpdateClients(srcState.updateHeaders, addr)...)
		}
		out.Dst = append(out.Dst, dst.Path().ChanConfirm(srcState.channel, addr))
		out.Last = true
	default:
		panic(fmt.Sprintf("not implemented error: %v <=> %v", srcState.channel.Channel.State.String(), dstState.channel.Channel.State.String()))
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

func querySettledChannel(
	queryCtx QueryContext,
	logger *log.RelayLogger,
	chain interface {
		Chain
		StateProver
	},
	prove bool,
) (*chantypes.QueryChannelResponse, bool, error) {
	ch, err := QueryChannel(queryCtx, chain, prove)
	if err != nil {
		logger.ErrorContext(queryCtx.Context(), "failed to query channel at the latest finalized height", err)
		return nil, false, err
	}

	var latestCtx QueryContext
	if h, err := chain.LatestHeight(queryCtx.Context()); err != nil {
		logger.ErrorContext(queryCtx.Context(), "failed to get the latest height", err)
		return nil, false, err
	} else {
		latestCtx = NewQueryContext(queryCtx.Context(), h)
	}

	latestChan, err := QueryChannel(latestCtx, chain, false)
	if err != nil {
		logger.ErrorContext(queryCtx.Context(), "failed to query channel at the latest height", err)
		return nil, false, err
	}

	if ch.Channel.String() != latestChan.Channel.String() {
		logger.DebugContext(queryCtx.Context(), "channel end in transition",
			"from", ch.Channel.String(),
			"to", latestChan.Channel.String(),
		)
		return ch, false, nil
	}
	return ch, true, nil
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
	srcChan, srcSettled, err := querySettledChannel(srcCtx, logger, src, prove)
	if err != nil {
		return nil, nil, false, err
	}
	dstChan, dstSettled, err := querySettledChannel(dstCtx, logger, dst, prove)
	if err != nil {
		return nil, nil, false, err
	}
	return srcChan, dstChan, (srcSettled && dstSettled), nil
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

func GetChannelPairLoggerRelative(me, cp Chain) *log.RelayLogger {
	return log.GetLogger().
		WithChannelPairRelative(
			me.ChainID(), me.Path().PortID, me.Path().ChannelID,
			cp.ChainID(), cp.Path().PortID, cp.Path().ChannelID,
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
