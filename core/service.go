package core

import (
	"context"
	"time"

	retry "github.com/avast/retry-go"
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
	sh, err := NewSyncHeaders(src, dst)
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
			select {
			case <-ctx.Done():
				return retry.Unrecoverable(ctx.Err())
			default:
				return srv.Serve(ctx)
			}
		}, rtyAtt, rtyDel, rtyErr, retry.OnRetry(func(n uint, err error) {
			logger.Info(
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
		time.Sleep(srv.interval)
	}
}

// Serve performs packet-relay
func (srv *RelayService) Serve(ctx context.Context) error {
	logger := GetChannelPairLogger(srv.src, srv.dst)

	// First, update the latest headers for src and dst
	if err := srv.sh.Updates(srv.src, srv.dst); err != nil {
		logger.Error("failed to update headers", err)
		return err
	}

	// get unrelayed packets
	pseqs, err := srv.st.UnrelayedPackets(srv.src, srv.dst, srv.sh, false)
	if err != nil {
		logger.Error("failed to get unrelayed packets", err)
		return err
	}

	// get unrelayed acks
	aseqs, err := srv.st.UnrelayedAcknowledgements(srv.src, srv.dst, srv.sh, false)
	if err != nil {
		logger.Error("failed to get unrelayed acknowledgements", err)
		return err
	}

	msgs := NewRelayMsgs()

	doExecuteRelaySrc, doExecuteRelayDst := srv.shouldExecuteRelay(pseqs)
	doExecuteAckSrc, doExecuteAckDst := srv.shouldExecuteRelay(aseqs)
	// update clients
	if m, err := srv.st.UpdateClients(srv.src, srv.dst, doExecuteRelaySrc, doExecuteRelayDst, doExecuteAckSrc, doExecuteAckDst, srv.sh, true); err != nil {
		logger.Error("failed to update clients", err)
		return err
	} else {
		msgs.Merge(m)
	}

	// relay packets if unrelayed seqs exist
	if m, err := srv.st.RelayPackets(srv.src, srv.dst, pseqs, srv.sh, doExecuteRelaySrc, doExecuteRelayDst); err != nil {
		logger.Error("failed to relay packets", err)
		return err
	} else {
		msgs.Merge(m)
	}

	// relay acks if unrelayed seqs exist
	if m, err := srv.st.RelayAcknowledgements(srv.src, srv.dst, aseqs, srv.sh, doExecuteAckSrc, doExecuteAckDst); err != nil {
		logger.Error("failed to relay acknowledgements", err)
		return err
	} else {
		msgs.Merge(m)
	}

	// send all msgs to src/dst chains
	srv.st.Send(context.TODO(), srv.src, srv.dst, msgs)

	return nil
}

func (srv *RelayService) shouldExecuteRelay(seqs *RelayPackets) (bool, bool) {
	logger := GetChannelPairLogger(srv.src, srv.dst)

	srcRelay := false
	dstRelay := false

	if len(seqs.Src) > 0 {
		tsDst, err := srv.src.Timestamp(context.TODO(), seqs.Src[0].EventHeight)
		if err != nil {
			return false, false
		}
		if time.Since(tsDst) >= srv.optimizeRelay.dstOptimizeInterval {
			dstRelay = true
		}
	}

	if len(seqs.Dst) > 0 {
		tsSrc, err := srv.dst.Timestamp(context.TODO(), seqs.Dst[0].EventHeight)
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

	logger.Info("shouldExecuteRelay", "srcRelay", srcRelay, "dstRelay", dstRelay)

	return srcRelay, dstRelay
}
