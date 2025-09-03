package core

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"time"

	retry "github.com/avast/retry-go"
	sdk "github.com/cosmos/cosmos-sdk/types"
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

type createChannelFutureProofs struct {
	updateHeaders []Header
	chanRes       *chantypes.QueryChannelResponse
}

type createChannelFutureMsg = func(proofs *createChannelFutureProofs) (msg []sdk.Msg, last bool)

func resolveCreateChannelFutureProofs(
	ctx context.Context,
	sh SyncHeaders,
	from, to *ProvableChain,
	fromProofs *createChannelFutureProofs,
) error {
	logger := GetChannelPairLoggerRelative(from, to)
	queryCtx := sh.GetQueryContext(ctx, from.ChainID())
	var err error

	fromProofs.updateHeaders, err = sh.SetupHeadersForUpdate(ctx, from, to)
	if err != nil {
		logger.ErrorContext(ctx, "error setting up headers for update", err)
		return err
	}

	if fromProofs.chanRes != nil {
		err = ProveChannel(queryCtx, from, fromProofs.chanRes)
		if err != nil {
			return err
		}
	}
	return nil
}

type createChannelFutureMsgs struct {
	Src, Dst []createChannelFutureMsg
}

func resolveCreateChannelFutureMsgs(
	ctx context.Context,
	sh SyncHeaders,
	fmsgs *createChannelFutureMsgs,
	src, dst *ProvableChain,
	srcProofs, dstProofs *createChannelFutureProofs,
) (*RelayMsgs, error) {
	ret := NewRelayMsgs()

	err := retry.Do(func() error {
		var eg = new(errgroup.Group)

		eg.Go(func() error { // send to Dst
			err := resolveCreateChannelFutureProofs(ctx, sh, src, dst, srcProofs)
			if err != nil {
				return err
			}

			for _, fm := range fmsgs.Dst {
				msgs, last := fm(srcProofs)
				ret.Dst = append(ret.Dst, msgs...)
				ret.Last = ret.Last || last
			}
			return nil
		})

		eg.Go(func() error { // send to Src
			err := resolveCreateChannelFutureProofs(ctx, sh, dst, src, dstProofs)
			if err != nil {
				return err
			}

			for _, fm := range fmsgs.Src {
				msgs, last := fm(dstProofs)
				ret.Src = append(ret.Src, msgs...)
				ret.Last = ret.Last || last
			}
			return nil
		})
		return eg.Wait()
	}, rtyAtt, rtyDel, rtyErr, retry.Context(ctx), retry.OnRetry(func(n uint, err error) {
	}))
	if err != nil {
		return nil, err
	}

	if len(dstProofs.updateHeaders) > 0 {
		srcAddr := mustGetAddress(src)
		ret.Src = append(src.Path().UpdateClients(dstProofs.updateHeaders, srcAddr), ret.Src...)
	}
	if len(srcProofs.updateHeaders) > 0 {
		dstAddr := mustGetAddress(dst)
		ret.Dst = append(dst.Path().UpdateClients(srcProofs.updateHeaders, dstAddr), ret.Dst...)
	}

	return ret, nil
}

func createChannelStep(ctx context.Context, src, dst *ProvableChain) (*RelayMsgs, error) {
	fmsgs := createChannelFutureMsgs{}
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
		srcProofs, dstProofs createChannelFutureProofs
	)

	var settled bool
	srcProofs.chanRes, dstProofs.chanRes, settled, err = querySettledChannelPair(
		sh.GetQueryContext(ctx, src.ChainID()),
		sh.GetQueryContext(ctx, dst.ChainID()),
		src,
		dst,
		false,
	)
	if err != nil {
		return nil, err
	} else if !settled {
		return NewRelayMsgs(), nil
	}

	switch {
	// Handshake hasn't been started on src or dst, relay `chanOpenInit` to src
	case srcProofs.chanRes.Channel.State == chantypes.UNINITIALIZED && dstProofs.chanRes.Channel.State == chantypes.UNINITIALIZED:
		fmsgs.Src = append(fmsgs.Src, func(p *createChannelFutureProofs) ([]sdk.Msg, bool) {
			logChannelStates(ctx, src, dst, srcProofs.chanRes, dstProofs.chanRes)
			msgs := make([]sdk.Msg, 0, 2)
			addr := mustGetAddress(src)
			msgs = append(msgs, src.Path().ChanInit(dst.Path(), addr))
			return msgs, false
		})
	// Handshake has started on dst (1 step done), relay `chanOpenTry` and `updateClient` to src
	case srcProofs.chanRes.Channel.State == chantypes.UNINITIALIZED && dstProofs.chanRes.Channel.State == chantypes.INIT:
		fmsgs.Src = append(fmsgs.Src, func(p *createChannelFutureProofs) ([]sdk.Msg, bool) {
			logChannelStates(ctx, src, dst, srcProofs.chanRes, dstProofs.chanRes)
			msgs := make([]sdk.Msg, 0, 2)
			addr := mustGetAddress(src)
			if len(p.updateHeaders) > 0 {
				msgs = append(msgs, src.Path().UpdateClients(p.updateHeaders, addr)...)
			}
			msgs = append(msgs, src.Path().ChanTry(dst.Path(), p.chanRes, addr))
			return msgs, false
		})
	// Handshake has started on src (1 step done), relay `chanOpenTry` and `updateClient` to dst
	case srcProofs.chanRes.Channel.State == chantypes.INIT && dstProofs.chanRes.Channel.State == chantypes.UNINITIALIZED:
		fmsgs.Dst = append(fmsgs.Dst, func(p *createChannelFutureProofs) ([]sdk.Msg, bool) {
			logChannelStates(ctx, dst, src, dstProofs.chanRes, srcProofs.chanRes)
			msgs := make([]sdk.Msg, 0, 2)
			addr := mustGetAddress(dst)
			if len(p.updateHeaders) > 0 {
				msgs = append(msgs, dst.Path().UpdateClients(p.updateHeaders, addr)...)
			}
			msgs = append(msgs, dst.Path().ChanTry(src.Path(), p.chanRes, addr))
			return msgs, false
		})

	// Handshake has started on src (2 steps done), relay `chanOpenAck` and `updateClient` to dst
	case srcProofs.chanRes.Channel.State == chantypes.TRYOPEN && dstProofs.chanRes.Channel.State == chantypes.INIT:
		fmsgs.Dst = append(fmsgs.Dst, func(p *createChannelFutureProofs) ([]sdk.Msg, bool) {
			logChannelStates(ctx, dst, src, dstProofs.chanRes, srcProofs.chanRes)
			msgs := make([]sdk.Msg, 0, 2)
			addr := mustGetAddress(dst)
			if len(p.updateHeaders) > 0 {
				msgs = append(msgs, dst.Path().UpdateClients(p.updateHeaders, addr)...)
			}
			msgs = append(msgs, dst.Path().ChanAck(src.Path(), p.chanRes, addr))
			return msgs, false
		})

	// Handshake has started on dst (2 steps done), relay `chanOpenAck` and `updateClient` to src
	case srcProofs.chanRes.Channel.State == chantypes.INIT && dstProofs.chanRes.Channel.State == chantypes.TRYOPEN:
		fmsgs.Src = append(fmsgs.Src, func(p *createChannelFutureProofs) ([]sdk.Msg, bool) {
			logChannelStates(ctx, src, dst, srcProofs.chanRes, dstProofs.chanRes)
			msgs := make([]sdk.Msg, 0, 2)
			addr := mustGetAddress(src)
			if len(p.updateHeaders) > 0 {
				msgs = append(msgs, src.Path().UpdateClients(p.updateHeaders, addr)...)
			}
			msgs = append(msgs, src.Path().ChanAck(dst.Path(), p.chanRes, addr))
			return msgs, false
		})

	// Handshake has confirmed on dst (3 steps done), relay `chanOpenConfirm` and `updateClient` to src
	case srcProofs.chanRes.Channel.State == chantypes.TRYOPEN && dstProofs.chanRes.Channel.State == chantypes.OPEN:
		fmsgs.Src = append(fmsgs.Src, func(p *createChannelFutureProofs) ([]sdk.Msg, bool) {
			logChannelStates(ctx, src, dst, srcProofs.chanRes, dstProofs.chanRes)
			msgs := make([]sdk.Msg, 0, 2)
			addr := mustGetAddress(src)
			if len(p.updateHeaders) > 0 {
				msgs = append(msgs, src.Path().UpdateClients(p.updateHeaders, addr)...)
			}
			msgs = append(msgs, src.Path().ChanConfirm(p.chanRes, addr))
			return msgs, true
		})

	// Handshake has confirmed on src (3 steps done), relay `chanOpenConfirm` and `updateClient` to dst
	case srcProofs.chanRes.Channel.State == chantypes.OPEN && dstProofs.chanRes.Channel.State == chantypes.TRYOPEN:
		fmsgs.Dst = append(fmsgs.Dst, func(p *createChannelFutureProofs) ([]sdk.Msg, bool) {
			logChannelStates(ctx, dst, src, dstProofs.chanRes, srcProofs.chanRes)
			msgs := make([]sdk.Msg, 0, 2)
			addr := mustGetAddress(dst)
			if len(p.updateHeaders) > 0 {
				msgs = append(msgs, dst.Path().UpdateClients(p.updateHeaders, addr)...)
			}
			msgs = append(msgs, dst.Path().ChanConfirm(p.chanRes, addr))
			return msgs, true
		})
	default:
		panic(fmt.Sprintf("not implemeneted error: %v <=> %v", srcProofs.chanRes.Channel.State.String(), dstProofs.chanRes.Channel.State.String()))
	}

	out, err := resolveCreateChannelFutureMsgs(ctx, sh, &fmsgs, src, dst, &srcProofs, &dstProofs)
	if err != nil {
		return nil, err
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
	return WithChannelPairAttributesAndKey("src", src, "dst", dst)
}

func WithChannelPairAttributesAndKey(srcKey string, src Chain, dstKey string, dst Chain) trace.SpanStartOption {
	return trace.WithAttributes(slices.Concat(
		semconv.AttributeGroup(srcKey,
			semconv.ChainIDKey.String(src.ChainID()),
			semconv.PortIDKey.String(src.Path().PortID),
			semconv.ChannelIDKey.String(src.Path().ChannelID),
		),
		semconv.AttributeGroup(dstKey,
			semconv.ChainIDKey.String(dst.ChainID()),
			semconv.PortIDKey.String(dst.Path().PortID),
			semconv.ChannelIDKey.String(dst.Path().ChannelID),
		),
	)...)
}
