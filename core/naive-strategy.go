package core

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	retry "github.com/avast/retry-go"
	sdk "github.com/cosmos/cosmos-sdk/types"
	chantypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
	"github.com/hyperledger-labs/yui-relayer/internal/telemetry"
	"github.com/hyperledger-labs/yui-relayer/otelcore/semconv"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	api "go.opentelemetry.io/otel/metric"
	"golang.org/x/sync/errgroup"
)

// NaiveStrategy is an implementation of Strategy.
type NaiveStrategy struct {
	Ordered      bool
	MaxTxSize    uint64 // maximum permitted size of the msgs in a bundled relay transaction
	MaxMsgLength uint64 // maximum amount of messages in a bundled relay transaction
	srcNoAck     bool
	dstNoAck     bool

	metrics naiveStrategyMetrics
}

type naiveStrategyMetrics struct {
	srcBacklog PacketInfoList
	dstBacklog PacketInfoList
}

var _ StrategyI = (*NaiveStrategy)(nil)

func NewNaiveStrategy(srcNoAck, dstNoAck bool) *NaiveStrategy {
	return &NaiveStrategy{
		srcNoAck: srcNoAck,
		dstNoAck: dstNoAck,
	}
}

// GetType implements Strategy
func (st *NaiveStrategy) GetType() string {
	return "naive"
}

func (st *NaiveStrategy) SetupRelay(ctx context.Context, src, dst *ProvableChain) error {
	logger := GetChannelPairLogger(src, dst)
	if err := src.SetupForRelay(ctx); err != nil {
		logger.ErrorContext(ctx, "failed to setup for src", err)
		return err
	}
	if err := dst.SetupForRelay(ctx); err != nil {
		logger.ErrorContext(ctx, "failed to setup for dst", err)
		return err
	}
	return nil
}

func getQueryContext(ctx context.Context, chain *ProvableChain, sh SyncHeaders, useFinalizedHeader bool) (QueryContext, error) {
	if useFinalizedHeader {
		return sh.GetQueryContext(ctx, chain.ChainID()), nil
	} else {
		height, err := chain.LatestHeight(ctx)
		if err != nil {
			return nil, err
		}
		return NewQueryContext(ctx, height), nil
	}
}

func (st *NaiveStrategy) UnrelayedPackets(ctx context.Context, src, dst *ProvableChain, sh SyncHeaders, includeRelayedButUnfinalized bool) (*RelayPackets, error) {
	ctx, span := tracer.Start(ctx, "NaiveStrategy.UnrelayedPackets", WithChannelPairAttributes(src, dst))
	defer span.End()
	logger := GetChannelPairLogger(src, dst)
	now := time.Now()
	var (
		eg         = new(errgroup.Group)
		srcPackets PacketInfoList
		dstPackets PacketInfoList
	)

	srcCtx, err := getQueryContext(ctx, src, sh, true)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	dstCtx, err := getQueryContext(ctx, dst, sh, true)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	eg.Go(func() error {
		return retry.Do(func() error {
			var err error
			now := time.Now()
			srcPackets, err = src.QueryUnfinalizedRelayPackets(srcCtx, dst)
			if err != nil {
				return fmt.Errorf("failed to query unfinalized relay packets on src chain: %w", err)
			}
			logger.TimeTrackContext(ctx, now, "QueryUnfinalizedRelayPackets", "queried_chain", "src", "num_packets", len(srcPackets))
			return nil
		}, rtyAtt, rtyDel, rtyErr, retry.Context(ctx), retry.OnRetry(func(n uint, err error) {
			logger.InfoContext(ctx,
				"retrying to query unfinalized packet relays",
				"direction", "src",
				"height", srcCtx.Height().GetRevisionHeight(),
				"try", n+1,
				"try_limit", rtyAttNum,
				"error", err.Error(),
			)
		}))
	})

	eg.Go(func() error {
		return retry.Do(func() error {
			var err error
			now := time.Now()
			dstPackets, err = dst.QueryUnfinalizedRelayPackets(dstCtx, src)
			if err != nil {
				return fmt.Errorf("failed to query unfinalized relay packets on dst chain: %w", err)
			}
			logger.TimeTrackContext(ctx, now, "QueryUnfinalizedRelayPackets", "queried_chain", "dst", "num_packets", len(dstPackets))
			return nil
		}, rtyAtt, rtyDel, rtyErr, retry.Context(ctx), retry.OnRetry(func(n uint, err error) {
			logger.InfoContext(ctx,
				"retrying to query unfinalized packet relays",
				"direction", "dst",
				"height", dstCtx.Height().GetRevisionHeight(),
				"try", n+1,
				"try_limit", rtyAttNum,
				"error", err.Error(),
			)
		}))
	})

	if err := eg.Wait(); err != nil {
		logger.ErrorContext(ctx, "error querying packet commitments", err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	if err := st.metrics.updateBacklogMetrics(ctx, src, dst, srcPackets, dstPackets); err != nil {
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	// If includeRelayedButUnfinalized is true, this function should return packets of which RecvPacket is not finalized yet.
	// In this case, filtering packets by QueryUnreceivedPackets is not needed because QueryUnfinalizedRelayPackets
	// has already returned packets that completely match this condition.
	if !includeRelayedButUnfinalized {
		srcCtx, err := getQueryContext(ctx, src, sh, false)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}
		dstCtx, err := getQueryContext(ctx, dst, sh, false)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}

		eg.Go(func() error {
			now := time.Now()
			seqs, err := dst.QueryUnreceivedPackets(dstCtx, srcPackets.ExtractSequenceList())
			if err != nil {
				return fmt.Errorf("failed to query unreceived packets on dst chain: %w", err)
			}
			logger.TimeTrackContext(ctx, now, "QueryUnreceivedPackets", "queried_chain", "dst", "num_seqs", len(seqs))
			srcPackets = srcPackets.Filter(seqs)
			return nil
		})

		eg.Go(func() error {
			now := time.Now()
			seqs, err := src.QueryUnreceivedPackets(srcCtx, dstPackets.ExtractSequenceList())
			if err != nil {
				return fmt.Errorf("failed to query unreceived packets on src chain: %w", err)
			}
			logger.TimeTrackContext(ctx, now, "QueryUnreceivedPackets", "queried_chain", "src", "num_seqs", len(seqs))
			dstPackets = dstPackets.Filter(seqs)
			return nil
		})

		if err := eg.Wait(); err != nil {
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}
	}

	defer logger.TimeTrackContext(ctx, now, "UnrelayedPackets", "num_src", len(srcPackets), "num_dst", len(dstPackets))

	return &RelayPackets{
		Src: srcPackets,
		Dst: dstPackets,
	}, nil
}

func (st *NaiveStrategy) RelayPackets(ctx context.Context, src, dst *ProvableChain, rp *RelayPackets, sh SyncHeaders, doExecuteRelaySrc, doExecuteRelayDst bool) (*RelayMsgs, error) {
	ctx, span := tracer.Start(ctx, "NaiveStrategy.RelayPackets", WithChannelPairAttributes(src, dst))
	defer span.End()
	logger := GetChannelPairLogger(src, dst)
	defer logger.TimeTrackContext(ctx, time.Now(), "RelayPackets", "num_src", len(rp.Src), "num_dst", len(rp.Dst))

	msgs := NewRelayMsgs()

	srcCtx := sh.GetQueryContext(ctx, src.ChainID())
	dstCtx := sh.GetQueryContext(ctx, dst.ChainID())
	srcAddress, err := src.GetAddress()
	if err != nil {
		logger.ErrorContext(ctx, "error getting address", err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	dstAddress, err := dst.GetAddress()
	if err != nil {
		logger.ErrorContext(ctx, "error getting address", err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	if doExecuteRelayDst {
		msgs.Dst, err = collectPackets(srcCtx, src, rp.Src, dstAddress)
		if err != nil {
			logger.ErrorContext(ctx, "error collecting packets", err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}
	}

	if doExecuteRelaySrc {
		msgs.Src, err = collectPackets(dstCtx, dst, rp.Dst, srcAddress)
		if err != nil {
			logger.ErrorContext(ctx, "error collecting packets", err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}
	}

	if len(msgs.Dst) == 0 && len(msgs.Src) == 0 {
		logger.InfoContext(ctx, "no packates to relay")
	} else {
		if num := len(msgs.Dst); num > 0 {
			logPacketsRelayed(ctx, src, dst, num, "Packets", "src->dst")
		}
		if num := len(msgs.Src); num > 0 {
			logPacketsRelayed(ctx, src, dst, num, "Packets", "dst->src")
		}
	}

	return msgs, nil
}

func (st *NaiveStrategy) UnrelayedAcknowledgements(ctx context.Context, src, dst *ProvableChain, sh SyncHeaders, includeRelayedButUnfinalized bool) (*RelayPackets, error) {
	ctx, span := tracer.Start(ctx, "NaiveStrategy.UnrelayedAcknowledgements", WithChannelPairAttributes(src, dst))
	defer span.End()
	logger := GetChannelPairLogger(src, dst)
	now := time.Now()
	var (
		eg      = new(errgroup.Group)
		srcAcks PacketInfoList
		dstAcks PacketInfoList
	)

	srcCtx, err := getQueryContext(ctx, src, sh, true)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	dstCtx, err := getQueryContext(ctx, dst, sh, true)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	if !st.dstNoAck {
		eg.Go(func() error {
			return retry.Do(func() error {
				var err error
				now := time.Now()
				srcAcks, err = src.QueryUnfinalizedRelayAcknowledgements(srcCtx, dst)
				if err != nil {
					return fmt.Errorf("failed to query unfinalized relay acknowledgements on src chain: %w", err)
				}
				logger.TimeTrackContext(ctx, now, "QueryUnfinalizedRelayAcknowledgements", "queried_chain", "src", "num_packets", len(srcAcks))
				return nil
			}, rtyAtt, rtyDel, rtyErr, retry.Context(ctx), retry.OnRetry(func(n uint, err error) {
				logger.InfoContext(ctx,
					"retrying to query unfinalized ack relays",
					"direction", "src",
					"height", srcCtx.Height().GetRevisionHeight(),
					"try", n+1,
					"try_limit", rtyAttNum,
					"error", err.Error(),
				)
				sh.Updates(ctx, src, dst)
			}))
		})
	}

	if !st.srcNoAck {
		eg.Go(func() error {
			return retry.Do(func() error {
				var err error
				now := time.Now()
				dstAcks, err = dst.QueryUnfinalizedRelayAcknowledgements(dstCtx, src)
				if err != nil {
					return fmt.Errorf("failed to query unfinalized relay acknowledgements on dst chain: %w", err)
				}
				logger.TimeTrackContext(ctx, now, "QueryUnfinalizedRelayAcknowledgements", "queried_chain", "dst", "num_packets", len(dstAcks))
				return nil
			}, rtyAtt, rtyDel, rtyErr, retry.Context(ctx), retry.OnRetry(func(n uint, err error) {
				logger.InfoContext(ctx,
					"retrying to query unfinalized ack relays",
					"direction", "dst",
					"height", dstCtx.Height().GetRevisionHeight(),
					"try", n+1,
					"try_limit", rtyAttNum,
					"error", err.Error(),
				)
				sh.Updates(ctx, src, dst)
			}))
		})
	}

	if err := eg.Wait(); err != nil {
		logger.ErrorContext(ctx, "error querying packet commitments", err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	// If includeRelayedButUnfinalized is true, this function should return packets of which AcknowledgePacket is not finalized yet.
	// In this case, filtering packets by QueryUnreceivedAcknowledgements is not needed because QueryUnfinalizedRelayAcknowledgements
	// has already returned packets that completely match this condition.
	if !includeRelayedButUnfinalized {
		srcCtx, err := getQueryContext(ctx, src, sh, false)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}
		dstCtx, err := getQueryContext(ctx, dst, sh, false)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}

		if !st.dstNoAck {
			eg.Go(func() error {
				now := time.Now()
				seqs, err := dst.QueryUnreceivedAcknowledgements(dstCtx, srcAcks.ExtractSequenceList())
				if err != nil {
					return fmt.Errorf("failed to query unreceived acknowledgements on dst chain: %w", err)
				}
				logger.TimeTrackContext(ctx, now, "QueryUnreceivedAcknowledgements", "queried_chain", "dst", "num_seqs", len(seqs))
				srcAcks = srcAcks.Filter(seqs)
				return nil
			})
		}

		if !st.srcNoAck {
			eg.Go(func() error {
				now := time.Now()
				seqs, err := src.QueryUnreceivedAcknowledgements(srcCtx, dstAcks.ExtractSequenceList())
				if err != nil {
					return fmt.Errorf("failed to query unreceived acknowledgements on src chain: %w", err)
				}
				logger.TimeTrackContext(ctx, now, "QueryUnreceivedAcknowledgements", "queried_chain", "src", "num_seqs", len(seqs))
				dstAcks = dstAcks.Filter(seqs)
				return nil
			})
		}

		if err := eg.Wait(); err != nil {
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}
	}

	defer logger.TimeTrackContext(ctx, now, "UnrelayedAcknowledgements", "num_src", len(srcAcks), "num_dst", len(dstAcks))

	return &RelayPackets{
		Src: srcAcks,
		Dst: dstAcks,
	}, nil
}

// TODO add packet-timeout support
func collectPackets(ctx QueryContext, chain *ProvableChain, packets PacketInfoList, signer sdk.AccAddress) ([]sdk.Msg, error) {
	logger := GetChannelLogger(chain)
	var msgs []sdk.Msg
	for _, p := range packets {
		commitment := chantypes.CommitPacket(chain.Codec(), &p.Packet)
		path := host.PacketCommitmentPath(p.SourcePort, p.SourceChannel, p.Sequence)
		proof, proofHeight, err := chain.ProveState(ctx, path, commitment)
		if err != nil {
			logger.ErrorContext(ctx.Context(), "failed to ProveState", err,
				"height", ctx.Height(),
				"path", path,
				"commitment", commitment,
			)
			return nil, err
		}
		msg := chantypes.NewMsgRecvPacket(p.Packet, proof, proofHeight, signer.String())
		msgs = append(msgs, msg)
	}
	return msgs, nil
}

func logPacketsRelayed(ctx context.Context, src, dst Chain, num int, obj, dir string) {
	logger := GetChannelPairLogger(src, dst)
	logger.InfoContext(ctx,
		fmt.Sprintf("★ %s are scheduled for relay", obj),
		"count", num,
		"direction", dir,
	)
}

func (st *NaiveStrategy) RelayAcknowledgements(ctx context.Context, src, dst *ProvableChain, rp *RelayPackets, sh SyncHeaders, doExecuteAckSrc, doExecuteAckDst bool) (*RelayMsgs, error) {
	ctx, span := tracer.Start(ctx, "NaiveStrategy.RelayAcknowledgements", WithChannelPairAttributes(src, dst))
	defer span.End()
	logger := GetChannelPairLogger(src, dst)
	defer logger.TimeTrackContext(ctx, time.Now(), "RelayAcknowledgements", "num_src", len(rp.Src), "num_dst", len(rp.Dst))

	msgs := NewRelayMsgs()

	srcCtx := sh.GetQueryContext(ctx, src.ChainID())
	dstCtx := sh.GetQueryContext(ctx, dst.ChainID())
	srcAddress, err := src.GetAddress()
	if err != nil {
		logger.ErrorContext(ctx, "error getting address", err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	dstAddress, err := dst.GetAddress()
	if err != nil {
		logger.ErrorContext(ctx, "error getting address", err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	if !st.dstNoAck && doExecuteAckDst {
		msgs.Dst, err = collectAcks(srcCtx, src, rp.Src, dstAddress)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}
	}
	if !st.srcNoAck && doExecuteAckSrc {
		msgs.Src, err = collectAcks(dstCtx, dst, rp.Dst, srcAddress)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}
	}

	if len(msgs.Dst) == 0 && len(msgs.Src) == 0 {
		logger.InfoContext(ctx, "no acknowledgements to relay")
	} else {
		if num := len(msgs.Dst); num > 0 {
			logPacketsRelayed(ctx, src, dst, num, "Acknowledgements", "src->dst")
		}
		if num := len(msgs.Src); num > 0 {
			logPacketsRelayed(ctx, src, dst, num, "Acknowledgements", "dst->src")
		}
	}

	return msgs, nil
}

func collectAcks(ctx QueryContext, chain *ProvableChain, packets PacketInfoList, signer sdk.AccAddress) ([]sdk.Msg, error) {
	logger := GetChannelLogger(chain)
	var msgs []sdk.Msg

	for _, p := range packets {
		commitment := chantypes.CommitAcknowledgement(p.Acknowledgement)
		path := host.PacketAcknowledgementPath(p.DestinationPort, p.DestinationChannel, p.Sequence)
		proof, proofHeight, err := chain.ProveState(ctx, path, commitment)
		if err != nil {
			logger.ErrorContext(ctx.Context(), "failed to ProveState", err,
				"height", ctx.Height(),
				"path", path,
				"commitment", commitment,
			)
			return nil, err
		}
		msg := chantypes.NewMsgAcknowledgement(p.Packet, p.Acknowledgement, proof, proofHeight, signer.String())
		msgs = append(msgs, msg)
	}

	return msgs, nil
}

func (st *NaiveStrategy) UpdateClients(ctx context.Context, src, dst *ProvableChain, doExecuteRelaySrc, doExecuteRelayDst, doExecuteAckSrc, doExecuteAckDst bool, sh SyncHeaders, doRefresh bool) (*RelayMsgs, error) {
	ctx, span := tracer.Start(ctx, "NaiveStrategy.UpdateClients", WithChannelPairAttributes(src, dst))
	defer span.End()
	logger := GetChannelPairLogger(src, dst)

	msgs := NewRelayMsgs()

	needsUpdateForSrc := doExecuteRelaySrc || (doExecuteAckSrc && !st.srcNoAck)
	needsUpdateForDst := doExecuteRelayDst || (doExecuteAckDst && !st.dstNoAck)

	// check if LC refresh is needed
	if !needsUpdateForSrc && doRefresh {
		var err error
		needsUpdateForSrc, err = dst.CheckRefreshRequired(ctx, src)
		if err != nil {
			err = fmt.Errorf("failed to check if the LC on the src chain needs to be refreshed: %v", err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}
	}
	if !needsUpdateForDst && doRefresh {
		var err error
		needsUpdateForDst, err = src.CheckRefreshRequired(ctx, dst)
		if err != nil {
			err = fmt.Errorf("failed to check if the LC on the dst chain needs to be refreshed: %v", err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}
	}

	if needsUpdateForSrc {
		srcAddress, err := src.GetAddress()
		if err != nil {
			err = fmt.Errorf("failed to get relayer address on src chain: %v", err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}
		hs, err := sh.SetupHeadersForUpdate(ctx, dst, src)
		if err != nil {
			err = fmt.Errorf("failed to set up headers for updating client on src chain: %v", err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}
		if len(hs) > 0 {
			msgs.Src = src.Path().UpdateClients(hs, srcAddress)
		}
	}

	if needsUpdateForDst {
		dstAddress, err := dst.GetAddress()
		if err != nil {
			err = fmt.Errorf("failed to get relayer address on dst chain: %v", err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}
		hs, err := sh.SetupHeadersForUpdate(ctx, src, dst)
		if err != nil {
			err = fmt.Errorf("failed to set up headers for updating client on dst chain: %v", err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}
		if len(hs) > 0 {
			msgs.Dst = dst.Path().UpdateClients(hs, dstAddress)
		}
	}

	if len(msgs.Src) > 0 {
		logger.InfoContext(ctx, "client on src chain was scheduled for update", "num_sent_msgs", len(msgs.Src))
	}
	if len(msgs.Dst) > 0 {
		logger.InfoContext(ctx, "client on dst chain was scheduled for update", "num_sent_msgs", len(msgs.Dst))
	}

	return msgs, nil
}

func (st *NaiveStrategy) Send(ctx context.Context, src, dst Chain, msgs *RelayMsgs) {
	ctx, span := tracer.Start(ctx, "NaiveStrategy.Send", WithChannelPairAttributes(src, dst))
	defer span.End()
	logger := GetChannelPairLogger(src, dst)

	msgs.MaxTxSize = st.MaxTxSize
	msgs.MaxMsgLength = st.MaxMsgLength
	msgs.Send(ctx, src, dst)

	logger.InfoContext(ctx, "msgs relayed",
		slog.Group("src", "msg_count", len(msgs.Src)),
		slog.Group("dst", "msg_count", len(msgs.Dst)),
	)
}

func (st *naiveStrategyMetrics) updateBacklogMetrics(ctx context.Context, src, dst ChainInfo, newSrcBacklog, newDstBacklog PacketInfoList) error {
	srcAttrs := []attribute.KeyValue{
		semconv.ChainIDKey.String(src.ChainID()),
		semconv.DirectionKey.String("src"),
	}
	dstAttrs := []attribute.KeyValue{
		semconv.ChainIDKey.String(dst.ChainID()),
		semconv.DirectionKey.String("dst"),
	}

	telemetry.BacklogSizeGauge.Set(int64(len(newSrcBacklog)), srcAttrs...)
	telemetry.BacklogSizeGauge.Set(int64(len(newDstBacklog)), dstAttrs...)

	if len(newSrcBacklog) > 0 {
		oldestHeight := newSrcBacklog[0].EventHeight
		oldestTimestamp, err := src.Timestamp(ctx, oldestHeight)
		if err != nil {
			return fmt.Errorf("failed to get the timestamp of block[%d] on the src chain: %v", oldestHeight, err)
		}
		telemetry.BacklogOldestTimestampGauge.Set(oldestTimestamp.UnixNano(), srcAttrs...)
	} else {
		telemetry.BacklogOldestTimestampGauge.Set(0, srcAttrs...)
	}
	if len(newDstBacklog) > 0 {
		oldestHeight := newDstBacklog[0].EventHeight
		oldestTimestamp, err := dst.Timestamp(ctx, oldestHeight)
		if err != nil {
			return fmt.Errorf("failed to get the timestamp of block[%d] on the dst chain: %v", oldestHeight, err)
		}
		telemetry.BacklogOldestTimestampGauge.Set(oldestTimestamp.UnixNano(), dstAttrs...)
	} else {
		telemetry.BacklogOldestTimestampGauge.Set(0, dstAttrs...)
	}

	srcReceivedPackets := st.srcBacklog.Subtract(newSrcBacklog.ExtractSequenceList())
	telemetry.ReceivePacketsFinalizedCounter.Add(ctx, int64(len(srcReceivedPackets)), api.WithAttributes(srcAttrs...))
	st.srcBacklog = newSrcBacklog

	dstReceivedPackets := st.dstBacklog.Subtract(newDstBacklog.ExtractSequenceList())
	telemetry.ReceivePacketsFinalizedCounter.Add(ctx, int64(len(dstReceivedPackets)), api.WithAttributes(dstAttrs...))
	st.dstBacklog = newDstBacklog

	return nil
}
