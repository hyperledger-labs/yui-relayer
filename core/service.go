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
	relayInterval, relayOptimizeInterval time.Duration,
	relayOptimizeCount int64) error {
	sh, err := NewSyncHeaders(src, dst)
	if err != nil {
		return err
	}
	srv := NewRelayService(st, src, dst, sh, relayInterval, relayOptimizeInterval, relayOptimizeCount)
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
	optimizeInterval time.Duration
	optimizeCount    int64

	srcRelayPacketStartTime WatchStartTime
	dstRelayPacketStartTime WatchStartTime
	srcRelayAckStartTime    WatchStartTime
	dstRelayAckStartTime    WatchStartTime
}

type WatchStartTime struct {
	Reset     bool
	StartTime time.Time
}

// NewRelayService returns a new service
func NewRelayService(
	st StrategyI,
	src, dst *ProvableChain,
	sh SyncHeaders,
	interval, optimizeInterval time.Duration,
	optimizeCount int64) *RelayService {
	return &RelayService{
		src:      src,
		dst:      dst,
		st:       st,
		sh:       sh,
		interval: interval,
		optimizeRelay: OptimizeRelay{
			optimizeInterval: optimizeInterval,
			optimizeCount:    optimizeCount,
			srcRelayPacketStartTime: WatchStartTime{
				Reset:     false,
				StartTime: time.Now(),
			},
			dstRelayPacketStartTime: WatchStartTime{
				Reset:     false,
				StartTime: time.Now(),
			},
			srcRelayAckStartTime: WatchStartTime{
				Reset:     false,
				StartTime: time.Now(),
			},
			dstRelayAckStartTime: WatchStartTime{
				Reset:     false,
				StartTime: time.Now(),
			},
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
		logger.Error("failed to get unrelayed sequences", err)
		return err
	}

	// get unrelayed acks
	aseqs, err := srv.st.UnrelayedAcknowledgements(srv.src, srv.dst, srv.sh, false)
	if err != nil {
		logger.Error("failed to get unrelayed acknowledgements", err)
		return err
	}

	msgs := NewRelayMsgs()

	// update clients
	if m, err := srv.st.UpdateClients(srv.src, srv.dst, pseqs, aseqs, srv.sh, true); err != nil {
		logger.Error("failed to update clients", err)
		return err
	} else {
		msgs.Merge(m)
	}

	// reset watch start time for packets
	if len(pseqs.Src) > 0 {
		resetWatchStartTime(&srv.optimizeRelay.srcRelayPacketStartTime)
	}
	if len(pseqs.Dst) > 0 {
		resetWatchStartTime(&srv.optimizeRelay.dstRelayPacketStartTime)
	}

	// relay packets if unrelayed seqs exist
	srcRelayPackets, dstRelayPackets := srv.shouldExecuteRelay(pseqs, srv.optimizeRelay.srcRelayPacketStartTime, srv.optimizeRelay.dstRelayPacketStartTime)
	srv.st.SkipRelay(!srcRelayPackets, !dstRelayPackets)
	if m, err := srv.st.RelayPackets(srv.src, srv.dst, pseqs, srv.sh); err != nil {
		logger.Error("failed to relay packets", err)
		return err
	} else {
		if srcRelayPackets {
			srv.optimizeRelay.srcRelayPacketStartTime.Reset = true
		}
		if dstRelayPackets {
			srv.optimizeRelay.dstRelayPacketStartTime.Reset = true
		}
		msgs.Merge(m)
	}

	// reset watch start time for acks
	if len(aseqs.Src) > 0 {
		resetWatchStartTime(&srv.optimizeRelay.srcRelayAckStartTime)
	}
	if len(aseqs.Dst) > 0 {
		resetWatchStartTime(&srv.optimizeRelay.dstRelayAckStartTime)
	}

	// relay acks if unrelayed seqs exist
	srcRelayAcks, dstRelayAcks := srv.shouldExecuteRelay(aseqs, srv.optimizeRelay.srcRelayAckStartTime, srv.optimizeRelay.dstRelayAckStartTime)
	srv.st.SkipRelay(!srcRelayAcks, !dstRelayAcks)
	if m, err := srv.st.RelayAcknowledgements(srv.src, srv.dst, aseqs, srv.sh); err != nil {
		logger.Error("failed to relay acknowledgements", err)
		return err
	} else {
		if srcRelayAcks {
			srv.optimizeRelay.srcRelayAckStartTime.Reset = true
		}
		if dstRelayAcks {
			srv.optimizeRelay.dstRelayAckStartTime.Reset = true
		}
		msgs.Merge(m)
	}

	// send all msgs to src/dst chains
	srv.st.Send(srv.src, srv.dst, msgs)

	return nil
}

func (srv *RelayService) shouldExecuteRelay(seqs *RelayPackets, srcRelayStartTime, dstRelayStartTime WatchStartTime) (bool, bool) {
	srcRelay := false
	dstRelay := false

	// packet count
	srcRelayCount := len(seqs.Src)
	dstRelayCount := len(seqs.Dst)
	if int64(srcRelayCount) >= srv.optimizeRelay.optimizeCount {
		srcRelay = true
	}
	if int64(dstRelayCount) >= srv.optimizeRelay.optimizeCount {
		dstRelay = true
	}

	// time interval
	if time.Since(srcRelayStartTime.StartTime) >= srv.optimizeRelay.optimizeInterval {
		srcRelay = true
	}
	if time.Since(dstRelayStartTime.StartTime) >= srv.optimizeRelay.optimizeInterval {
		dstRelay = true
	}

	return srcRelay, dstRelay
}

func resetWatchStartTime(watchStartTime *WatchStartTime) {
	if !watchStartTime.Reset {
		return
	}
	watchStartTime.Reset = false
	watchStartTime.StartTime = time.Now()
}
