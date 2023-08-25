package core

import (
	"context"
	"fmt"

	retry "github.com/avast/retry-go"
	sdk "github.com/cosmos/cosmos-sdk/types"
	chantypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v7/modules/core/24-host"
	"github.com/hyperledger-labs/yui-relayer/log"
	"golang.org/x/sync/errgroup"
)

// NaiveStrategy is an implementation of Strategy.
type NaiveStrategy struct {
	Ordered      bool
	MaxTxSize    uint64 // maximum permitted size of the msgs in a bundled relay transaction
	MaxMsgLength uint64 // maximum amount of messages in a bundled relay transaction
	srcNoAck     bool
	dstNoAck     bool
}

var _ StrategyI = (*NaiveStrategy)(nil)

func NewNaiveStrategy(srcNoAck, dstNoAck bool) *NaiveStrategy {
	return &NaiveStrategy{
		srcNoAck: srcNoAck,
		dstNoAck: dstNoAck,
	}
}

// GetType implements Strategy
func (st NaiveStrategy) GetType() string {
	return "naive"
}

func (st NaiveStrategy) SetupRelay(ctx context.Context, src, dst *ProvableChain) error {
	logger := GetChannelLogger(log.GetLogger(), src, dst)
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

func getQueryContext(chain *ProvableChain, sh SyncHeaders, useFinalizedHeader bool) (QueryContext, error) {
	if useFinalizedHeader {
		return sh.GetQueryContext(chain.ChainID()), nil
	} else {
		height, err := chain.LatestHeight()
		if err != nil {
			return nil, err
		}
		return NewQueryContext(context.TODO(), height), nil
	}
}

func (st NaiveStrategy) UnrelayedPackets(src, dst *ProvableChain, sh SyncHeaders, includeRelayedButUnfinalized bool) (*RelayPackets, error) {
	logger := GetChannelLogger(log.GetLogger(), src, dst)
	var (
		eg         = new(errgroup.Group)
		srcPackets PacketInfoList
		dstPackets PacketInfoList
	)

	srcCtx, err := getQueryContext(src, sh, true)
	if err != nil {
		return nil, err
	}
	dstCtx, err := getQueryContext(dst, sh, true)
	if err != nil {
		return nil, err
	}

	eg.Go(func() error {
		return retry.Do(func() error {
			var err error
			srcPackets, err = src.QueryUnfinalizedRelayPackets(srcCtx, dst)
			return err
		}, rtyAtt, rtyDel, rtyErr, retry.OnRetry(func(n uint, err error) {
			logger.Info(
				"query unfinalized packets",
				"src revision height", srcCtx.Height().GetRevisionHeight(),
				"try", fmt.Sprintf("%d/%d", n+1, rtyAttNum),
				"error", err.Error(),
			)
		}))
	})

	eg.Go(func() error {
		return retry.Do(func() error {
			var err error
			dstPackets, err = dst.QueryUnfinalizedRelayPackets(dstCtx, src)
			return err
		}, rtyAtt, rtyDel, rtyErr, retry.OnRetry(func(n uint, err error) {
			logger.Info(
				"query unfinalized packets",
				"dst revision height", dstCtx.Height().GetRevisionHeight(),
				"try", fmt.Sprintf("%d/%d", n+1, rtyAttNum),
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

	// If includeRelayedButUnfinalized is true, this function should return packets of which RecvPacket is not finalized yet.
	// In this case, filtering packets by QueryUnreceivedPackets is not needed because QueryUnfinalizedRelayPackets
	// has already returned packets that completely match this condition.
	if !includeRelayedButUnfinalized {
		srcCtx, err := getQueryContext(src, sh, false)
		if err != nil {
			return nil, err
		}
		dstCtx, err := getQueryContext(dst, sh, false)
		if err != nil {
			return nil, err
		}

		eg.Go(func() error {
			seqs, err := dst.QueryUnreceivedPackets(dstCtx, srcPackets.ExtractSequenceList())
			if err != nil {
				return err
			}
			srcPackets = srcPackets.Filter(seqs)
			return nil
		})

		eg.Go(func() error {
			seqs, err := src.QueryUnreceivedPackets(srcCtx, dstPackets.ExtractSequenceList())
			if err != nil {
				return err
			}
			dstPackets = dstPackets.Filter(seqs)
			return nil
		})

		if err := eg.Wait(); err != nil {
			return nil, err
		}
	}

	return &RelayPackets{
		Src: srcPackets,
		Dst: dstPackets,
	}, nil
}

func (st NaiveStrategy) RelayPackets(src, dst *ProvableChain, rp *RelayPackets, sh SyncHeaders) error {
	logger := GetChannelLogger(log.GetLogger(), src, dst)
	// set the maximum relay transaction constraints
	msgs := &RelayMsgs{
		Src:          []sdk.Msg{},
		Dst:          []sdk.Msg{},
		MaxTxSize:    st.MaxTxSize,
		MaxMsgLength: st.MaxMsgLength,
	}

	srcCtx := sh.GetQueryContext(src.ChainID())
	dstCtx := sh.GetQueryContext(dst.ChainID())
	srcAddress, err := src.GetAddress()
	if err != nil {
		logger.Error(
			"error getting address",
			err,
		)
		return err
	}

	dstAddress, err := dst.GetAddress()
	if err != nil {
		logger.Error(
			"error getting address",
			err,
		)
		return err
	}

	if len(rp.Src) > 0 {
		hs, err := sh.SetupHeadersForUpdate(src, dst)
		if err != nil {
			logger.Error(
				"error setting up headers for update",
				err,
			)
			return err
		}
		if len(hs) > 0 {
			msgs.Dst = dst.Path().UpdateClients(hs, dstAddress)
		}
	}

	if len(rp.Dst) > 0 {
		hs, err := sh.SetupHeadersForUpdate(dst, src)
		if err != nil {
			logger.Error(
				"error setting up headers for update",
				err,
			)
			return err
		}
		if len(hs) > 0 {
			msgs.Src = src.Path().UpdateClients(hs, srcAddress)
		}
	}

	packetsForDst, err := collectPackets(srcCtx, src, rp.Src, dstAddress)
	if err != nil {
		logger.Error(
			"error collecting packets",
			err,
		)
		return err
	}
	packetsForSrc, err := collectPackets(dstCtx, dst, rp.Dst, srcAddress)
	if err != nil {
		logger.Error(
			"error collecting packets",
			err,
		)
		return err
	}

	if len(packetsForDst) == 0 && len(packetsForSrc) == 0 {
		logger.Info(
			"no packates to relay",
		)
		return nil
	}

	msgs.Dst = append(msgs.Dst, packetsForDst...)
	msgs.Src = append(msgs.Src, packetsForSrc...)

	// send messages to their respective chains
	if msgs.Send(src, dst); msgs.Success() {
		if num := len(packetsForDst); num > 0 {
			logPacketsRelayed(dst, num, "RelayPackets")
		}
		if num := len(packetsForSrc); num > 0 {
			logPacketsRelayed(src, num, "RelayPackets")
		}
	}

	return nil
}

func (st NaiveStrategy) UnrelayedAcknowledgements(src, dst *ProvableChain, sh SyncHeaders, includeRelayedButUnfinalized bool) (*RelayPackets, error) {
	logger := GetChannelLogger(log.GetLogger(), src, dst)
	var (
		eg      = new(errgroup.Group)
		srcAcks PacketInfoList
		dstAcks PacketInfoList
	)

	srcCtx, err := getQueryContext(src, sh, true)
	if err != nil {
		return nil, err
	}
	dstCtx, err := getQueryContext(dst, sh, true)
	if err != nil {
		return nil, err
	}

	if !st.dstNoAck {
		eg.Go(func() error {
			return retry.Do(func() error {
				var err error
				srcAcks, err = src.QueryUnfinalizedRelayAcknowledgements(srcCtx, dst)
				return err
			}, rtyAtt, rtyDel, rtyErr, retry.OnRetry(func(n uint, err error) {
				logger.Info(
					"query packet acknowledgements",
					"src revision height", srcCtx.Height().GetRevisionHeight(),
					"try", fmt.Sprintf("%d/%d", n+1, rtyAttNum),
					"error", err.Error(),
				)
				sh.Updates(src, dst)
			}))
		})
	}

	if !st.srcNoAck {
		eg.Go(func() error {
			return retry.Do(func() error {
				var err error
				dstAcks, err = dst.QueryUnfinalizedRelayAcknowledgements(dstCtx, src)
				return err
			}, rtyAtt, rtyDel, rtyErr, retry.OnRetry(func(n uint, err error) {
				logger.Info(
					"query packet acknowledgements",
					"dst revision height", dstCtx.Height().GetRevisionHeight(),
					"try", fmt.Sprintf("%d/%d", n+1, rtyAttNum),
					"error", err.Error(),
				)
				sh.Updates(src, dst)
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
		srcCtx, err := getQueryContext(src, sh, false)
		if err != nil {
			return nil, err
		}
		dstCtx, err := getQueryContext(dst, sh, false)
		if err != nil {
			return nil, err
		}

		if !st.dstNoAck {
			eg.Go(func() error {
				seqs, err := dst.QueryUnreceivedAcknowledgements(dstCtx, srcAcks.ExtractSequenceList())
				if err != nil {
					return err
				}
				srcAcks = srcAcks.Filter(seqs)
				return nil
			})
		}

		if !st.srcNoAck {
			eg.Go(func() error {
				seqs, err := src.QueryUnreceivedAcknowledgements(srcCtx, dstAcks.ExtractSequenceList())
				if err != nil {
					return err
				}
				dstAcks = dstAcks.Filter(seqs)
				return nil
			})
		}

		if err := eg.Wait(); err != nil {
			return nil, err
		}
	}

	return &RelayPackets{
		Src: srcAcks,
		Dst: dstAcks,
	}, nil
}

// TODO add packet-timeout support
func collectPackets(ctx QueryContext, chain *ProvableChain, packets PacketInfoList, signer sdk.AccAddress) ([]sdk.Msg, error) {
	relayLogger := log.GetLogger()
	var msgs []sdk.Msg
	for _, p := range packets {
		commitment := chantypes.CommitPacket(chain.Codec(), &p.Packet)
		path := host.PacketCommitmentPath(p.SourcePort, p.SourceChannel, p.Sequence)
		proof, proofHeight, err := chain.ProveState(ctx, path, commitment)
		if err != nil {
			relayLogger.Info("failed to ProveState: height=%v, path=%v, commitment=%x, err=%v", ctx.Height(), path, commitment, err)
			return nil, err
		}
		msg := chantypes.NewMsgRecvPacket(p.Packet, proof, proofHeight, signer.String())
		msgs = append(msgs, msg)
	}
	return msgs, nil
}

func logPacketsRelayed(targetChain *ProvableChain, num int, msg string) {
	relayLogger := log.GetLogger()
	relayLogger.Info(
		fmt.Sprintf("★ Relayed %d packets", num),
		"chain id", targetChain.ChainID(),
		"channel id", targetChain.Path().ChannelID,
		"port id", targetChain.Path().PortID,
		"msg", msg,
	)
}

func (st NaiveStrategy) RelayAcknowledgements(src, dst *ProvableChain, rp *RelayPackets, sh SyncHeaders) error {
	logger := GetChannelLogger(log.GetLogger(), src, dst)
	// set the maximum relay transaction constraints
	msgs := &RelayMsgs{
		Src:          []sdk.Msg{},
		Dst:          []sdk.Msg{},
		MaxTxSize:    st.MaxTxSize,
		MaxMsgLength: st.MaxMsgLength,
	}

	srcCtx := sh.GetQueryContext(src.ChainID())
	dstCtx := sh.GetQueryContext(dst.ChainID())
	srcAddress, err := src.GetAddress()
	if err != nil {
		logger.Error(
			"error getting address",
			err,
		)
		return err
	}
	dstAddress, err := dst.GetAddress()
	if err != nil {
		logger.Error(
			"error getting address",
			err,
		)
		return err
	}

	if !st.dstNoAck && len(rp.Src) > 0 {
		hs, err := sh.SetupHeadersForUpdate(src, dst)
		if err != nil {
			logger.Error(
				"error setting up headers",
				err,
			)
			return err
		}
		if len(hs) > 0 {
			msgs.Dst = dst.Path().UpdateClients(hs, dstAddress)
		}
	}

	if !st.srcNoAck && len(rp.Dst) > 0 {
		hs, err := sh.SetupHeadersForUpdate(dst, src)
		if err != nil {
			logger.Error(
				"error setting up headers",
				err,
			)
			return err
		}
		if len(hs) > 0 {
			msgs.Src = src.Path().UpdateClients(hs, srcAddress)
		}
	}

	var acksForSrc, acksForDst []sdk.Msg
	if !st.dstNoAck {
		acksForDst, err = collectAcks(srcCtx, src, rp.Src, dstAddress)
		if err != nil {
			return err
		}
	}
	if !st.srcNoAck {
		acksForSrc, err = collectAcks(dstCtx, dst, rp.Dst, srcAddress)
		if err != nil {
			return err
		}
	}

	if len(acksForDst) == 0 && len(acksForSrc) == 0 {
		logger.Info(
			"no acknowledgements to relay",
		)
		return nil
	}

	msgs.Dst = append(msgs.Dst, acksForDst...)
	msgs.Src = append(msgs.Src, acksForSrc...)

	// send messages to their respective chains
	if msgs.Send(src, dst); msgs.Success() {
		if num := len(acksForDst); num > 0 {
			logPacketsRelayed(dst, num, "RelayAcknowledgements")
		}
		if num := len(acksForSrc); num > 0 {
			logPacketsRelayed(src, num, "RelayAcknowledgements")
		}
	}

	return nil
}

func collectAcks(ctx QueryContext, chain *ProvableChain, packets PacketInfoList, signer sdk.AccAddress) ([]sdk.Msg, error) {
	relayLogger := log.GetLogger()
	var msgs []sdk.Msg

	for _, p := range packets {
		commitment := chantypes.CommitAcknowledgement(p.Acknowledgement)
		path := host.PacketAcknowledgementPath(p.DestinationPort, p.DestinationChannel, p.Sequence)
		proof, proofHeight, err := chain.ProveState(ctx, path, commitment)
		if err != nil {
			relayLogger.Info("failed to ProveState: height=%v, path=%v, commitment=%x, err=%v", ctx.Height(), path, commitment, err)
			return nil, err
		}
		msg := chantypes.NewMsgAcknowledgement(p.Packet, p.Acknowledgement, proof, proofHeight, signer.String())
		msgs = append(msgs, msg)
	}

	return msgs, nil
}
