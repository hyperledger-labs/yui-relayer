package core

import (
	"context"
	"time"

	retry "github.com/avast/retry-go"
	"go.opentelemetry.io/otel/codes"
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

	pseqs2, err := srv.st.SortUnrelayedPackets(ctx, srv.src, srv.dst, srv.sh, pseqs)
	if err != nil {
		logger.Error("failed to sort unrelayed packets", err)
		return err
	}

	// get unrelayed acks
	aseqs, err := srv.st.UnrelayedAcknowledgements(ctx, srv.src, srv.dst, srv.sh, false)
	if err != nil {
		logger.ErrorContext(ctx, "failed to get unrelayed acknowledgements", err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	msgs := NewRelayMsgs()

	doExecuteRelaySrc, doExecuteRelayDst := srv.shouldExecuteRelay(ctx, pseqs2)
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
	if m, err := srv.st.RelayPackets(ctx, srv.src, srv.dst, pseqs2, srv.sh, doExecuteRelaySrc, doExecuteRelayDst); err != nil {
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
