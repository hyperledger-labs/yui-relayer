package core

import (
	"context"
	"time"
	"fmt"

	retry "github.com/avast/retry-go"
	"go.opentelemetry.io/otel/codes"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"golang.org/x/sync/errgroup"
)

// StartService starts a relay service
func StartService(
	ctx context.Context,
	st StrategyI,
	src, dst *ProvableChain,
	relayInterval,
	srcRelayOptimizeInterval time.Duration,
	srcRelayOptimizeCount uint64,
	dstRelayOptimizaInterval time.Duration,
	dstRelayOptimizeCount uint64,
) error {
	sh, err := NewSyncHeaders(ctx, src, dst)
	if err != nil {
		return err
	}
	srv := NewRelayService(
		st,
		src,
		dst,
		sh,
		relayInterval,
		srcRelayOptimizeInterval,
		srcRelayOptimizeCount,
		dstRelayOptimizaInterval,
		dstRelayOptimizeCount,
	)
	return srv.Start(ctx)
}

type RelayService struct {
	src           *ProvableChain
	dst           *ProvableChain
	st            StrategyI
	sh            SyncHeaders
	interval      time.Duration
	optimizeRelay OptimizeRelay
}

type OptimizeRelay struct {
	srcOptimizeInterval time.Duration
	srcOptimizeCount    uint64
	dstOptimizeInterval time.Duration
	dstOptimizeCount    uint64
}

// NewRelayService returns a new service
func NewRelayService(
	st StrategyI,
	src, dst *ProvableChain,
	sh SyncHeaders,
	interval,
	srcOptimizeInterval time.Duration,
	srcOptimizeCount uint64,
	dstOptimizeInterval time.Duration,
	dstOptimizeCount uint64,
) *RelayService {
	return &RelayService{
		src:      src,
		dst:      dst,
		st:       st,
		sh:       sh,
		interval: interval,
		optimizeRelay: OptimizeRelay{
			srcOptimizeInterval: srcOptimizeInterval,
			srcOptimizeCount:    srcOptimizeCount,
			dstOptimizeInterval: dstOptimizeInterval,
			dstOptimizeCount:    dstOptimizeCount,
		},
	}
}

// Start starts a relay service
func (srv *RelayService) Start(ctx context.Context) error {
	logger := GetChannelPairLogger(srv.src, srv.dst)
	for {
		if err := retry.Do(func() error {
			return srv.Serve(ctx)
		}, rtyAtt, rtyDel, rtyErr, retry.Context(ctx), retry.OnRetry(func(n uint, err error) {
			logger.InfoContext(ctx,
				"retrying to serve relays",
				"src", srv.src.ChainID(),
				"dst", srv.dst.ChainID(),
				"try", n+1,
				"try_limit", rtyAttNum,
				"error", err.Error(),
			)
		})); err != nil {
			return err
		}
		if err := wait(ctx, srv.interval); err != nil {
			return err
		}
	}
}

// Serve performs packet-relay
func (srv *RelayService) Serve(ctx context.Context) error {
	ctx, span := tracer.Start(ctx, "RelayService.Serve", WithChannelPairAttributes(srv.src, srv.dst))
	defer span.End()
	logger := GetChannelPairLogger(srv.src, srv.dst)

	// First, update the latest headers for src and dst
	if err := srv.sh.Updates(ctx, srv.src, srv.dst); err != nil {
		logger.ErrorContext(ctx, "failed to update headers", err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	// get unrelayed packets
	pseqs, err := srv.st.UnrelayedPackets(ctx, srv.src, srv.dst, srv.sh, false)
	if err != nil {
		logger.ErrorContext(ctx, "failed to get unrelayed packets", err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	// get unrelayed acks
	aseqs, err := srv.st.UnrelayedAcknowledgements(ctx, srv.src, srv.dst, srv.sh, false)
	if err != nil {
		logger.ErrorContext(ctx, "failed to get unrelayed acknowledgements", err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	relay := func(dir string, ctx context.Context, relayFrom, relayTo *ProvableChain, packets, acks PacketInfoList, sh SyncHeaders, doExecuteRelay, doExecuteAck, doRefresh bool) ([]sdk.Msg, error) {
		msgs := make([]sdk.Msg, 0, len(packets) + len(acks) + 1)

		// update clients
		if m, err := srv.st.UpdateClients(dir, ctx, relayFrom, relayTo, doExecuteRelay, doExecuteAck, sh, doRefresh); err != nil {
			logger.ErrorContext(ctx, "failed to update clients", err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		} else {
			msgs = append(msgs, m...)
		}

		// relay packets if unrelayed seqs exist
		if m, err := srv.st.RelayPackets(dir, ctx, relayFrom, relayTo, packets, sh, doExecuteRelay); err != nil {
			logger.ErrorContext(ctx, "failed to relay packets", err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		} else {
			msgs = append(msgs, m...)
		}

		// relay acks if unrelayed seqs exist
		if m, err := srv.st.RelayAcknowledgements(dir, ctx, relayFrom, relayTo, acks, sh, doExecuteAck); err != nil {
			logger.ErrorContext(ctx, "failed to relay acknowledgements", err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		} else {
			msgs = append(msgs, m...)
		}

		return msgs, nil
	}

	msgs := NewRelayMsgs()
	{
		var eg = new(errgroup.Group)
		srcStream := make(chan []sdk.Msg, 1)
		dstStream := make(chan []sdk.Msg, 1)
		defer close(srcStream)
		defer close(dstStream)

		doExecuteRelaySrc, doExecuteRelayDst := srv.shouldExecuteRelay(ctx, pseqs)
		doExecuteAckSrc, doExecuteAckDst := srv.shouldExecuteRelay(ctx, aseqs)
		eg.Go(func() error {
			msgs, err := relay("dst", ctx, srv.src, srv.dst, pseqs.Src, aseqs.Src, srv.sh, doExecuteRelayDst, doExecuteAckDst, true)
			if err != nil {
				return err
			}
			dstStream <- msgs
			return nil
		})
		eg.Go(func() error {
			msgs, err := relay("src", ctx, srv.dst, srv.src, pseqs.Dst, aseqs.Dst, srv.sh, doExecuteRelaySrc, doExecuteAckSrc, true)
			if err != nil {
				return err
			}
			srcStream <- msgs
			return nil
		})

		err := eg.Wait() // it waits querying to other chain. it may take more time and my chain's state is deleted.
		if err != nil {
			return err
		}
		var ok bool
		msgs.Dst, ok = <-dstStream
		if !ok {
			return fmt.Errorf("dstStream channel closed unexpectedly")
		}
		msgs.Src, ok = <-srcStream
		if !ok {
			return fmt.Errorf("srcStream channel closed unexpectedly")
		}
	}

/*
	msgs := NewRelayMsgs()

	doExecuteRelaySrc, doExecuteRelayDst := srv.shouldExecuteRelay(ctx, pseqs)
	doExecuteAckSrc, doExecuteAckDst := srv.shouldExecuteRelay(ctx, aseqs)
	// update clients
	if m, err := srv.st.UpdateClients(ctx, srv.src, srv.dst, doExecuteRelaySrc, doExecuteRelayDst, doExecuteAckSrc, doExecuteAckDst, srv.sh, true); err != nil {
		logger.ErrorContext(ctx, "failed to update clients", err)
		span.SetStatus(codes.Error, err.Error())
		return err
	} else {
		msgs.Merge(m)
	}

	// relay packets if unrelayed seqs exist
	if m, err := srv.st.RelayPackets(ctx, srv.src, srv.dst, pseqs, srv.sh, doExecuteRelaySrc, doExecuteRelayDst); err != nil {
		logger.ErrorContext(ctx, "failed to relay packets", err)
		span.SetStatus(codes.Error, err.Error())
		return err
	} else {
		msgs.Merge(m)
	}

	// relay acks if unrelayed seqs exist
	if m, err := srv.st.RelayAcknowledgements(ctx, srv.src, srv.dst, aseqs, srv.sh, doExecuteAckSrc, doExecuteAckDst); err != nil {
		logger.ErrorContext(ctx, "failed to relay acknowledgements", err)
		span.SetStatus(codes.Error, err.Error())
		return err
	} else {
		msgs.Merge(m)
	}
*/
	// send all msgs to src/dst chains
	srv.st.Send(ctx, srv.src, srv.dst, msgs)

	return nil
}

func (srv *RelayService) shouldExecuteRelay(ctx context.Context, seqs *RelayPackets) (bool, bool) {
	logger := GetChannelPairLogger(srv.src, srv.dst)

	srcRelay := false
	dstRelay := false

	if len(seqs.Src) > 0 {
		tsDst, err := srv.src.Timestamp(ctx, seqs.Src[0].EventHeight)
		if err != nil {
			return false, false
		}
		if time.Since(tsDst) >= srv.optimizeRelay.dstOptimizeInterval {
			dstRelay = true
		}
	}

	if len(seqs.Dst) > 0 {
		tsSrc, err := srv.dst.Timestamp(ctx, seqs.Dst[0].EventHeight)
		if err != nil {
			return false, false
		}
		if time.Since(tsSrc) >= srv.optimizeRelay.srcOptimizeInterval {
			srcRelay = true
		}
	}

	// packet count
	srcRelayCount := len(seqs.Dst)
	dstRelayCount := len(seqs.Src)

	if uint64(srcRelayCount) >= srv.optimizeRelay.srcOptimizeCount {
		srcRelay = true
	}
	if uint64(dstRelayCount) >= srv.optimizeRelay.dstOptimizeCount {
		dstRelay = true
	}

	logger.InfoContext(ctx, "shouldExecuteRelay", "srcRelay", srcRelay, "dstRelay", dstRelay)

	return srcRelay, dstRelay
}
