package core

import (
	"context"
	"time"

	retry "github.com/avast/retry-go"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/hyperledger-labs/yui-relayer/log"
	"go.opentelemetry.io/otel/codes"
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

func (srv *RelayService) relayMsgs(ctx context.Context, isSrcToDst bool, packets, acks PacketInfoList, sh SyncHeaders, doExecuteRelay, doExecuteAck, doExecuteTimeout, doRefresh bool) ([]sdk.Msg, error) {
	ctx, span := tracer.Start(ctx, "RelayService.relayMsgs", WithChannelPairAttributes(srv.src, srv.dst))
	defer span.End()

	var logger *log.RelayLogger
	if isSrcToDst {
		logger = GetChannelPairLoggerRelative(srv.src, srv.dst)
	} else {
		logger = GetChannelPairLoggerRelative(srv.dst, srv.src)
	}

	// relay packets if unrelayed seqs exist
	packetMsgs, err := srv.st.RelayPackets(ctx, srv.src, srv.dst, isSrcToDst, packets, sh, doExecuteRelay)
	if err != nil {
		logger.ErrorContext(ctx, "failed to relay packets", err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	// relay acks if unrelayed seqs exist
	ackMsgs, err := srv.st.RelayAcknowledgements(ctx, srv.src, srv.dst, isSrcToDst, acks, sh, doExecuteAck)
	if err != nil {
		logger.ErrorContext(ctx, "failed to relay acknowledgements", err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	// update clients
	// NOTE: this must be called after all proofs are generated, but the resulting msgs must precede them in the tx
	var msgs []sdk.Msg
	if m, err := srv.st.UpdateClients(ctx, srv.src, srv.dst, isSrcToDst, doExecuteRelay, doExecuteAck, doExecuteTimeout, sh, doRefresh); err != nil {
		logger.ErrorContext(ctx, "failed to update clients", err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	} else {
		msgs = append(msgs, m...)
	}
	msgs = append(msgs, packetMsgs...)
	msgs = append(msgs, ackMsgs...)

	return msgs, nil
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

	// process timeout packets - classifies timed out packets into SrcTimeout and DstTimeout
	if err := srv.st.ProcessTimeoutPackets(ctx, srv.src, srv.dst, srv.sh, pseqs); err != nil {
		logger.ErrorContext(ctx, "failed to process timeout packets", err)
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

	msgs := NewRelayMsgs()
	{
		var eg = new(errgroup.Group)

		doExecuteRelaySrc, doExecuteRelayDst := srv.shouldExecuteRelay(ctx, pseqs)
		doExecuteAckSrc, doExecuteAckDst := srv.shouldExecuteRelay(ctx, aseqs)
		doExecuteTimeoutSrc := len(pseqs.SrcTimeout) > 0
		doExecuteTimeoutDst := len(pseqs.DstTimeout) > 0

		// Process src→dst direction: builds messages for dst chain
		// - MsgRecvPacket for pseqs.Src (src→dst packets)
		// - MsgTimeout for pseqs.DstTimeout (dst's packets that timed out, sent back to dst)
		eg.Go(func() error {
			isSrcToDst := true
			// Collect timeout messages for dst's packets (DstTimeout → dst chain)
			var timeoutMsgs []sdk.Msg
			if doExecuteTimeoutDst {
				var err error
				timeoutMsgs, err = srv.st.RelayTimeoutPackets(ctx, srv.dst, srv.src, pseqs.DstTimeout, srv.sh, true)
				if err != nil {
					return err
				}
			}
			m, err := srv.relayMsgs(ctx, isSrcToDst, pseqs.Src, aseqs.Src, srv.sh, doExecuteRelayDst, doExecuteAckDst, doExecuteTimeoutDst, true)
			if err != nil {
				return err
			}
			msgs.Dst = append(m, timeoutMsgs...)
			return nil
		})

		// Process dst→src direction: builds messages for src chain
		// - MsgRecvPacket for pseqs.Dst (dst→src packets)
		// - MsgTimeout for pseqs.SrcTimeout (src's packets that timed out, sent back to src)
		eg.Go(func() error {
			isSrcToDst := false
			// Collect timeout messages for src's packets (SrcTimeout → src chain)
			var timeoutMsgs []sdk.Msg
			if doExecuteTimeoutSrc {
				var err error
				timeoutMsgs, err = srv.st.RelayTimeoutPackets(ctx, srv.src, srv.dst, pseqs.SrcTimeout, srv.sh, true)
				if err != nil {
					return err
				}
			}
			m, err := srv.relayMsgs(ctx, isSrcToDst, pseqs.Dst, aseqs.Dst, srv.sh, doExecuteRelaySrc, doExecuteAckSrc, doExecuteTimeoutSrc, true)
			if err != nil {
				return err
			}
			msgs.Src = append(m, timeoutMsgs...)
			return nil
		})

		if err := eg.Wait(); err != nil {
			return err
		}
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
