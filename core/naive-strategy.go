package core

import (
	"context"
	"fmt"
	"log/slog"
	"time"
	"encoding/binary"

	retry "github.com/avast/retry-go"
	sdk "github.com/cosmos/cosmos-sdk/types"
	ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"
	chantypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
	"github.com/hyperledger-labs/yui-relayer/metrics"
	"go.opentelemetry.io/otel/attribute"
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
		logger.Error(
			"failed to setup for src",
			err,
		)
		return err
	}
	if err := dst.SetupForRelay(ctx); err != nil {
		logger.Error(
			"failed to setup for dst",
			err,
		)
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
	logger := GetChannelPairLogger(src, dst)
	now := time.Now()
	var (
		eg         = new(errgroup.Group)
		srcPackets PacketInfoList
		dstPackets PacketInfoList
	)

	srcCtx, err := getQueryContext(ctx, src, sh, true)
	if err != nil {
		return nil, err
	}
	dstCtx, err := getQueryContext(ctx, dst, sh, true)
	if err != nil {
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
			logger.TimeTrack(now, "QueryUnfinalizedRelayPackets", "queried_chain", "src", "num_packets", len(srcPackets))
			return nil
		}, rtyAtt, rtyDel, rtyErr, retry.Context(ctx), retry.OnRetry(func(n uint, err error) {
			logger.Info(
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
			logger.TimeTrack(now, "QueryUnfinalizedRelayPackets", "queried_chain", "dst", "num_packets", len(dstPackets))
			return nil
		}, rtyAtt, rtyDel, rtyErr, retry.Context(ctx), retry.OnRetry(func(n uint, err error) {
			logger.Info(
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
		logger.Error(
			"error querying packet commitments",
			err,
		)
		return nil, err
	}

	if err := st.metrics.updateBacklogMetrics(ctx, src, dst, srcPackets, dstPackets); err != nil {
		return nil, err
	}

	// If includeRelayedButUnfinalized is true, this function should return packets of which RecvPacket is not finalized yet.
	// In this case, filtering packets by QueryUnreceivedPackets is not needed because QueryUnfinalizedRelayPackets
	// has already returned packets that completely match this condition.
	if !includeRelayedButUnfinalized {
		srcCtx, err := getQueryContext(ctx, src, sh, false)
		if err != nil {
			return nil, err
		}
		dstCtx, err := getQueryContext(ctx, dst, sh, false)
		if err != nil {
			return nil, err
		}

		eg.Go(func() error {
			now := time.Now()
			seqs, err := dst.QueryUnreceivedPackets(dstCtx, srcPackets.ExtractSequenceList())
			if err != nil {
				return fmt.Errorf("failed to query unreceived packets on dst chain: %w", err)
			}
			logger.TimeTrack(now, "QueryUnreceivedPackets", "queried_chain", "dst", "num_seqs", len(seqs))
			srcPackets = srcPackets.Filter(seqs)
			return nil
		})

		eg.Go(func() error {
			now := time.Now()
			seqs, err := src.QueryUnreceivedPackets(srcCtx, dstPackets.ExtractSequenceList())
			if err != nil {
				return fmt.Errorf("failed to query unreceived packets on src chain: %w", err)
			}
			logger.TimeTrack(now, "QueryUnreceivedPackets", "queried_chain", "src", "num_seqs", len(seqs))
			dstPackets = dstPackets.Filter(seqs)
			return nil
		})

		if err := eg.Wait(); err != nil {
			return nil, err
		}
	}

	defer logger.TimeTrack(now, "UnrelayedPackets", "num_src", len(srcPackets), "num_dst", len(dstPackets))

	return &RelayPackets{
		Src: srcPackets,
		Dst: dstPackets,
	}, nil
}

func (st *NaiveStrategy) SortUnrelayedPackets(ctx context.Context, src, dst *ProvableChain, sh SyncHeaders, rp *RelayPackets) (*RelayPackets, error) {
	logger := GetChannelPairLogger(src, dst)
	var (
		srcPackets   PacketInfoList
		dstPackets   PacketInfoList
		srcHeight    ibcexported.Height
		srcTimestamp uint64
		dstHeight    ibcexported.Height
		dstTimestamp uint64
	)

	if 0 < len(rp.Src) {
		dstHeight = sh.GetLatestFinalizedHeader(dst.ChainID()).GetHeight()
		timestamp, err := dst.Timestamp(context.TODO(), dstHeight)
		if err != nil {
			logger.Error("fail to get dst.Timestamp", err)
			return nil, err
		}
		dstTimestamp = uint64(timestamp.UnixNano())
	}
	if 0 < len(rp.Dst) {
		srcHeight = sh.GetLatestFinalizedHeader(src.ChainID()).GetHeight()
		timestamp, err := src.Timestamp(context.TODO(), srcHeight)
		if err != nil {
			logger.Error("fail to get src.Timestamp", err)
			return nil, err
		}
		srcTimestamp = uint64(timestamp.UnixNano())
	}

	for i, p := range rp.Src {
		if (!p.TimeoutHeight.IsZero() && p.TimeoutHeight.LTE(dstHeight)) ||
			(p.TimeoutTimestamp != 0 && p.TimeoutTimestamp <= dstTimestamp) {
			p.Sort = "timeout"
			if src.Path().GetOrder() == chantypes.ORDERED {
				if i == 0 {
					dstPackets = append(dstPackets, p)
				}
				break
			} else {
				dstPackets = append(dstPackets, p)
			}
		} else {
			srcPackets = append(srcPackets, p)
		}
	}
	for i, p := range rp.Dst {
		if (!p.TimeoutHeight.IsZero() && p.TimeoutHeight.LTE(srcHeight)) ||
			(p.TimeoutTimestamp != 0 && p.TimeoutTimestamp <= srcTimestamp) {
			p.Sort = "timeout"
			if dst.Path().GetOrder() == chantypes.ORDERED {
				if i == 0 {
					srcPackets = append(srcPackets, p)
				}
				break
			} else {
				srcPackets = append(srcPackets, p)
			}
		} else {
			dstPackets = append(dstPackets, p)
		}
	}
	return &RelayPackets{
		Src: srcPackets,
		Dst: dstPackets,
	}, nil
}

func (st *NaiveStrategy) RelayPackets(ctx context.Context, src, dst *ProvableChain, rp *RelayPackets, sh SyncHeaders, doExecuteRelaySrc, doExecuteRelayDst bool) (*RelayMsgs, error) {
	logger := GetChannelPairLogger(src, dst)
	defer logger.TimeTrack(time.Now(), "RelayPackets", "num_src", len(rp.Src), "num_dst", len(rp.Dst))

	msgs := NewRelayMsgs()

	srcCtx := sh.GetQueryContext(ctx, src.ChainID())
	dstCtx := sh.GetQueryContext(ctx, dst.ChainID())
	srcAddress, err := src.GetAddress()
	if err != nil {
		logger.Error(
			"error getting address",
			err,
		)
		return nil, err
	}

	dstAddress, err := dst.GetAddress()
	if err != nil {
		logger.Error(
			"error getting address",
			err,
		)
		return nil, err
	}

	if doExecuteRelayDst {
		msgs.Dst, err = collectPackets(srcCtx, src, rp.Src, dstAddress)
		if err != nil {
			logger.Error(
				"error collecting packets",
				err,
			)
			return nil, err
		}
	}

	if doExecuteRelaySrc {
		msgs.Src, err = collectPackets(dstCtx, dst, rp.Dst, srcAddress)
		if err != nil {
			logger.Error(
				"error collecting packets",
				err,
			)
			return nil, err
		}
	}

	if len(msgs.Dst) == 0 && len(msgs.Src) == 0 {
		logger.Info("no packates to relay")
	} else {
		if num := len(msgs.Dst); num > 0 {
			logPacketsRelayed(src, dst, num, "Packets", "src->dst")
		}
		if num := len(msgs.Src); num > 0 {
			logPacketsRelayed(src, dst, num, "Packets", "dst->src")
		}
	}

	return msgs, nil
}

func (st *NaiveStrategy) UnrelayedAcknowledgements(ctx context.Context, src, dst *ProvableChain, sh SyncHeaders, includeRelayedButUnfinalized bool) (*RelayPackets, error) {
	logger := GetChannelPairLogger(src, dst)
	now := time.Now()
	var (
		eg      = new(errgroup.Group)
		srcAcks PacketInfoList
		dstAcks PacketInfoList
	)

	srcCtx, err := getQueryContext(ctx, src, sh, true)
	if err != nil {
		return nil, err
	}
	dstCtx, err := getQueryContext(ctx, dst, sh, true)
	if err != nil {
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
				logger.TimeTrack(now, "QueryUnfinalizedRelayAcknowledgements", "queried_chain", "src", "num_packets", len(srcAcks))
				return nil
			}, rtyAtt, rtyDel, rtyErr, retry.Context(ctx), retry.OnRetry(func(n uint, err error) {
				logger.Info(
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
				logger.TimeTrack(now, "QueryUnfinalizedRelayAcknowledgements", "queried_chain", "dst", "num_packets", len(dstAcks))
				return nil
			}, rtyAtt, rtyDel, rtyErr, retry.Context(ctx), retry.OnRetry(func(n uint, err error) {
				logger.Info(
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
		logger.Error(
			"error querying packet commitments",
			err,
		)
		return nil, err
	}

	// If includeRelayedButUnfinalized is true, this function should return packets of which AcknowledgePacket is not finalized yet.
	// In this case, filtering packets by QueryUnreceivedAcknowledgements is not needed because QueryUnfinalizedRelayAcknowledgements
	// has already returned packets that completely match this condition.
	if !includeRelayedButUnfinalized {
		srcCtx, err := getQueryContext(ctx, src, sh, false)
		if err != nil {
			return nil, err
		}
		dstCtx, err := getQueryContext(ctx, dst, sh, false)
		if err != nil {
			return nil, err
		}

		if !st.dstNoAck {
			eg.Go(func() error {
				now := time.Now()
				seqs, err := dst.QueryUnreceivedAcknowledgements(dstCtx, srcAcks.ExtractSequenceList())
				if err != nil {
					return fmt.Errorf("failed to query unreceived acknowledgements on dst chain: %w", err)
				}
				logger.TimeTrack(now, "QueryUnreceivedAcknowledgements", "queried_chain", "dst", "num_seqs", len(seqs))
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
				logger.TimeTrack(now, "QueryUnreceivedAcknowledgements", "queried_chain", "src", "num_seqs", len(seqs))
				dstAcks = dstAcks.Filter(seqs)
				return nil
			})
		}

		if err := eg.Wait(); err != nil {
			return nil, err
		}
	}

	defer logger.TimeTrack(now, "UnrelayedAcknowledgements", "num_src", len(srcAcks), "num_dst", len(dstAcks))

	return &RelayPackets{
		Src: srcAcks,
		Dst: dstAcks,
	}, nil
}

// TODO add packet-timeout support
func collectPackets(ctx QueryContext, chain *ProvableChain, packets PacketInfoList, signer sdk.AccAddress) ([]sdk.Msg, error) {
	logger := GetChannelLogger(chain)

	var nextSequenceRecv uint64
	if chain.Path().GetOrder() == chantypes.ORDERED {
		for _, p := range packets {
			if p.Sort == "timeout" {
				res, err := chain.QueryNextSequenceReceive(ctx)
				if err != nil {
					logger.Error("failed to QueryNextSequenceReceive", err,
						"height", ctx.Height(),
					)
					return nil, err
				}
				nextSequenceRecv = res.NextSequenceReceive
				break
			}
		}
	} else {
		// nextSequenceRecv has no effect in unordered channel but ibc-go expect it is not zero.
		nextSequenceRecv = 1
	}

	var msgs []sdk.Msg
	for _, p := range packets {
		var msg sdk.Msg
		if p.Sort == "timeout" {
			// make path of original packet's destination port and channel
			var path string
			var commitment []byte
			if chain.Path().GetOrder() == chantypes.ORDERED {
				path = host.NextSequenceRecvPath(p.SourcePort, p.SourceChannel)
				commitment = make([]byte, 8)
				binary.BigEndian.PutUint64(commitment[0:], nextSequenceRecv)
			} else {
				path = host.PacketReceiptPath(p.SourcePort, p.SourceChannel, p.Sequence)
				commitment = []byte{} //ABSENSE
			}
			proof, proofHeight, err := chain.ProveState(ctx, path, commitment)
			if err != nil {
				logger.Error("failed to ProveState", err,
					"height", ctx.Height(),
					"path", path,
					"commitment", commitment,
				)
				return nil, err
			}
			msg = chantypes.NewMsgTimeout(p.Packet, nextSequenceRecv, proof, proofHeight, signer.String())
		} else {
			path := host.PacketCommitmentPath(p.SourcePort, p.SourceChannel, p.Sequence)
			commitment := chantypes.CommitPacket(chain.Codec(), &p.Packet)
			proof, proofHeight, err := chain.ProveState(ctx, path, commitment)
			if err != nil {
				logger.Error("failed to ProveState", err,
					"height", ctx.Height(),
					"path", path,
					"commitment", commitment,
				)
				return nil, err
			}
			msg = chantypes.NewMsgRecvPacket(p.Packet, proof, proofHeight, signer.String())
		}
		msgs = append(msgs, msg)
	}
	return msgs, nil
}

func logPacketsRelayed(src, dst Chain, num int, obj string, dir string) {
	logger := GetChannelPairLogger(src, dst)
	logger.Info(
		fmt.Sprintf("★ %s are scheduled for relay", obj),
		"count", num,
		"direction", dir,
	)
}

func (st *NaiveStrategy) RelayAcknowledgements(ctx context.Context, src, dst *ProvableChain, rp *RelayPackets, sh SyncHeaders, doExecuteAckSrc, doExecuteAckDst bool) (*RelayMsgs, error) {
	logger := GetChannelPairLogger(src, dst)
	defer logger.TimeTrack(time.Now(), "RelayAcknowledgements", "num_src", len(rp.Src), "num_dst", len(rp.Dst))

	msgs := NewRelayMsgs()

	srcCtx := sh.GetQueryContext(ctx, src.ChainID())
	dstCtx := sh.GetQueryContext(ctx, dst.ChainID())
	srcAddress, err := src.GetAddress()
	if err != nil {
		logger.Error(
			"error getting address",
			err,
		)
		return nil, err
	}
	dstAddress, err := dst.GetAddress()
	if err != nil {
		logger.Error(
			"error getting address",
			err,
		)
		return nil, err
	}

	if !st.dstNoAck && doExecuteAckDst {
		msgs.Dst, err = collectAcks(srcCtx, src, rp.Src, dstAddress)
		if err != nil {
			return nil, err
		}
	}
	if !st.srcNoAck && doExecuteAckSrc {
		msgs.Src, err = collectAcks(dstCtx, dst, rp.Dst, srcAddress)
		if err != nil {
			return nil, err
		}
	}

	if len(msgs.Dst) == 0 && len(msgs.Src) == 0 {
		logger.Info("no acknowledgements to relay")
	} else {
		if num := len(msgs.Dst); num > 0 {
			logPacketsRelayed(src, dst, num, "Acknowledgements", "src->dst")
		}
		if num := len(msgs.Src); num > 0 {
			logPacketsRelayed(src, dst, num, "Acknowledgements", "dst->src")
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
			logger.Error("failed to ProveState", err,
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
	logger := GetChannelPairLogger(src, dst)

	msgs := NewRelayMsgs()

	needsUpdateForSrc := doExecuteRelaySrc || (doExecuteAckSrc && !st.srcNoAck)
	needsUpdateForDst := doExecuteRelayDst || (doExecuteAckDst && !st.dstNoAck)

	// check if LC refresh is needed
	if !needsUpdateForSrc && doRefresh {
		var err error
		needsUpdateForSrc, err = dst.CheckRefreshRequired(ctx, src)
		if err != nil {
			return nil, fmt.Errorf("failed to check if the LC on the src chain needs to be refreshed: %v", err)
		}
	}
	if !needsUpdateForDst && doRefresh {
		var err error
		needsUpdateForDst, err = src.CheckRefreshRequired(ctx, dst)
		if err != nil {
			return nil, fmt.Errorf("failed to check if the LC on the dst chain needs to be refreshed: %v", err)
		}
	}

	if needsUpdateForSrc {
		srcAddress, err := src.GetAddress()
		if err != nil {
			return nil, fmt.Errorf("failed to get relayer address on src chain: %v", err)
		}
		hs, err := sh.SetupHeadersForUpdate(ctx, dst, src)
		if err != nil {
			return nil, fmt.Errorf("failed to set up headers for updating client on src chain: %v", err)
		}
		if len(hs) > 0 {
			msgs.Src = src.Path().UpdateClients(hs, srcAddress)
		}
	}

	if needsUpdateForDst {
		dstAddress, err := dst.GetAddress()
		if err != nil {
			return nil, fmt.Errorf("failed to get relayer address on dst chain: %v", err)
		}
		hs, err := sh.SetupHeadersForUpdate(ctx, src, dst)
		if err != nil {
			return nil, fmt.Errorf("failed to set up headers for updating client on dst chain: %v", err)
		}
		if len(hs) > 0 {
			msgs.Dst = dst.Path().UpdateClients(hs, dstAddress)
		}
	}

	if len(msgs.Src) > 0 {
		logger.Info("client on src chain was scheduled for update", "num_sent_msgs", len(msgs.Src))
	}
	if len(msgs.Dst) > 0 {
		logger.Info("client on dst chain was scheduled for update", "num_sent_msgs", len(msgs.Dst))
	}

	return msgs, nil
}

func (st *NaiveStrategy) Send(ctx context.Context, src, dst Chain, msgs *RelayMsgs) {
	logger := GetChannelPairLogger(src, dst)

	msgs.MaxTxSize = st.MaxTxSize
	msgs.MaxMsgLength = st.MaxMsgLength
	msgs.Send(ctx, src, dst)

	logger.Info("msgs relayed",
		slog.Group("src", "msg_count", len(msgs.Src)),
		slog.Group("dst", "msg_count", len(msgs.Dst)),
	)
}

func (st *naiveStrategyMetrics) updateBacklogMetrics(ctx context.Context, src, dst ChainInfo, newSrcBacklog, newDstBacklog PacketInfoList) error {
	srcAttrs := []attribute.KeyValue{
		attribute.Key("chain_id").String(src.ChainID()),
		attribute.Key("direction").String("src"),
	}
	dstAttrs := []attribute.KeyValue{
		attribute.Key("chain_id").String(dst.ChainID()),
		attribute.Key("direction").String("dst"),
	}

	metrics.BacklogSizeGauge.Set(int64(len(newSrcBacklog)), srcAttrs...)
	metrics.BacklogSizeGauge.Set(int64(len(newDstBacklog)), dstAttrs...)

	if len(newSrcBacklog) > 0 {
		oldestHeight := newSrcBacklog[0].EventHeight
		oldestTimestamp, err := src.Timestamp(ctx, oldestHeight)
		if err != nil {
			return fmt.Errorf("failed to get the timestamp of block[%d] on the src chain: %v", oldestHeight, err)
		}
		metrics.BacklogOldestTimestampGauge.Set(oldestTimestamp.UnixNano(), srcAttrs...)
	} else {
		metrics.BacklogOldestTimestampGauge.Set(0, srcAttrs...)
	}
	if len(newDstBacklog) > 0 {
		oldestHeight := newDstBacklog[0].EventHeight
		oldestTimestamp, err := dst.Timestamp(ctx, oldestHeight)
		if err != nil {
			return fmt.Errorf("failed to get the timestamp of block[%d] on the dst chain: %v", oldestHeight, err)
		}
		metrics.BacklogOldestTimestampGauge.Set(oldestTimestamp.UnixNano(), dstAttrs...)
	} else {
		metrics.BacklogOldestTimestampGauge.Set(0, dstAttrs...)
	}

	srcReceivedPackets := st.srcBacklog.Subtract(newSrcBacklog.ExtractSequenceList())
	metrics.ReceivePacketsFinalizedCounter.Add(ctx, int64(len(srcReceivedPackets)), api.WithAttributes(srcAttrs...))
	st.srcBacklog = newSrcBacklog

	dstReceivedPackets := st.dstBacklog.Subtract(newDstBacklog.ExtractSequenceList())
	metrics.ReceivePacketsFinalizedCounter.Add(ctx, int64(len(dstReceivedPackets)), api.WithAttributes(dstAttrs...))
	st.dstBacklog = newDstBacklog

	return nil
}
