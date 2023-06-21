package core

import (
	"context"
	"fmt"
	"log"

	retry "github.com/avast/retry-go"
	sdk "github.com/cosmos/cosmos-sdk/types"
	chantypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	"github.com/hyperledger-labs/yui-relayer/logger"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

// NaiveStrategy is an implementation of Strategy.
type NaiveStrategy struct {
	Ordered      bool
	MaxTxSize    uint64 // maximum permitted size of the msgs in a bundled relay transaction
	MaxMsgLength uint64 // maximum amount of messages in a bundled relay transaction
}

var _ StrategyI = (*NaiveStrategy)(nil)

func NewNaiveStrategy() *NaiveStrategy {
	return &NaiveStrategy{}
}

// GetType implements Strategy
func (st NaiveStrategy) GetType() string {
	return "naive"
}

func (st NaiveStrategy) SetupRelay(ctx context.Context, src, dst *ProvableChain) error {
	logger := logger.ZapLogger()
	defer logger.Sync()
	if err := src.SetupForRelay(ctx); err != nil {
		logger.Error(fmt.Sprintf("failed to setup src chain [%s]chan{%s}port{%s} -> [%s]chan{%s}port{%s}",
			src.ChainID(), src.Path().ChannelID, src.Path().PortID,
			dst.ChainID(), dst.Path().ChannelID, dst.Path().PortID),
			zap.Error(err))
		return err
	}
	if err := dst.SetupForRelay(ctx); err != nil {
		logger.Error(fmt.Sprintf("failed to setup dst chain [%s]chan{%s}port{%s} -> [%s]chan{%s}port{%s}",
			src.ChainID(), src.Path().ChannelID, src.Path().PortID,
			dst.ChainID(), dst.Path().ChannelID, dst.Path().PortID),
			zap.Error(err))
		return err
	}
	return nil
}

func (st NaiveStrategy) UnrelayedSequences(src, dst *ProvableChain, sh SyncHeaders) (*RelaySequences, error) {
	logger := logger.ZapLogger()
	defer logger.Sync()
	var (
		eg           = new(errgroup.Group)
		srcPacketSeq = []uint64{}
		dstPacketSeq = []uint64{}
		err          error
		rs           = &RelaySequences{Src: []uint64{}, Dst: []uint64{}}
	)
	srcCtx := sh.GetQueryContext(src.ChainID())
	dstCtx := sh.GetQueryContext(dst.ChainID())

	eg.Go(func() error {
		var res *chantypes.QueryPacketCommitmentsResponse
		if err = retry.Do(func() error {
			// Query the packet commitment
			res, err = src.QueryPacketCommitments(srcCtx, 0, 1000)
			switch {
			case err != nil:
				logger.Error(fmt.Sprintf("- [%s]@{%d} - error querying packet commitments",
					src.ChainID(), srcCtx.Height().GetRevisionHeight()), zap.Error(err))
				return err
			case res == nil:
				logger.Error(fmt.Sprintf("- [%s]@{%d} - nil packet commitments",
					src.ChainID(), srcCtx.Height().GetRevisionHeight()), zap.Error(err))
				return fmt.Errorf("no error on QueryPacketCommitments for %s, however response is nil", src.ChainID())
			default:
				return nil
			}
		}, rtyAtt, rtyDel, rtyErr, retry.OnRetry(func(n uint, err error) {
			logger.Info(fmt.Sprintf("- [%s]@{%d} - try(%d/%d) query packet commitments: %s",
				src.ChainID(), srcCtx.Height().GetRevisionHeight(), n+1, rtyAttNum, err))
		})); err != nil {
			logger.Error(fmt.Sprintf("- [%s]@{%d} - max retry exceeded querying packet commitments",
				src.ChainID(), srcCtx.Height().GetRevisionHeight()), zap.Error(err))
			return err
		}
		for _, pc := range res.Commitments {
			srcPacketSeq = append(srcPacketSeq, pc.Sequence)
		}
		return nil
	})

	eg.Go(func() error {
		var res *chantypes.QueryPacketCommitmentsResponse
		if err = retry.Do(func() error {
			res, err = dst.QueryPacketCommitments(dstCtx, 0, 1000)
			switch {
			case err != nil:
				logger.Error(fmt.Sprintf("- [%s]@{%d} - error querying packet commitments",
					dst.ChainID(), dstCtx.Height().GetRevisionHeight()), zap.Error(err))
				return err
			case res == nil:
				logger.Error(fmt.Sprintf("- [%s]@{%d} - nil packet commitments",
					dst.ChainID(), dstCtx.Height().GetRevisionHeight()), zap.Error(err))
				return fmt.Errorf("no error on QueryPacketCommitments for %s, however response is nil", dst.ChainID())
			default:
				return nil
			}
		}, rtyAtt, rtyDel, rtyErr, retry.OnRetry(func(n uint, err error) {
			logger.Info(fmt.Sprintf("- [%s]@{%d} - try(%d/%d) query packet commitments: %s",
				dst.ChainID(), dstCtx.Height().GetRevisionHeight(), n+1, rtyAttNum, err))
		})); err != nil {
			logger.Error(fmt.Sprintf("- [%s]@{%d} - max retry exceeded querying packet commitments",
				dst.ChainID(), dstCtx.Height().GetRevisionHeight()), zap.Error(err))
			return err
		}
		for _, pc := range res.Commitments {
			dstPacketSeq = append(dstPacketSeq, pc.Sequence)
		}
		return nil
	})

	if err := eg.Wait(); err != nil {
		logger.Error(fmt.Sprintf("- [%s]@{%d} - error querying packet commitments",
			src.ChainID(), srcCtx.Height().GetRevisionHeight()), zap.Error(err))
		return nil, err
	}

	eg.Go(func() error {
		// Query all packets sent by src that have been received by dst
		src, err := dst.QueryUnrecievedPackets(dstCtx, srcPacketSeq)
		if err != nil {
			logger.Error(fmt.Sprintf("- [%s]@{%d} - error querying unrelayed packets",
				dst.ChainID(), dstCtx.Height().GetRevisionHeight()), zap.Error(err))
			return err
		} else if src != nil {
			rs.Src = src
		}
		return nil
	})

	eg.Go(func() error {
		// Query all packets sent by dst that have been received by src
		dst, err := src.QueryUnrecievedPackets(srcCtx, dstPacketSeq)
		if err != nil {
			logger.Error(fmt.Sprintf("- [%s]@{%d} - error querying unrelayed packets",
				src.ChainID(), srcCtx.Height().GetRevisionHeight()), zap.Error(err))
			return err
		} else if dst != nil {
			rs.Dst = dst
		}
		return nil
	})

	if err := eg.Wait(); err != nil {
		logger.Error(fmt.Sprintf("- [%s]@{%d} - error querying unrelayed packets",
			src.ChainID(), srcCtx.Height().GetRevisionHeight()), zap.Error(err))
		return nil, err
	}

	return rs, nil
}

func (st NaiveStrategy) RelayPackets(src, dst *ProvableChain, sp *RelaySequences, sh SyncHeaders) error {
	logger := logger.ZapLogger()
	defer logger.Sync()
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
		logger.Error(fmt.Sprintf("- [%s]@{%d} - error getting address",
			src.ChainID(), srcCtx.Height().GetRevisionHeight()), zap.Error(err))
		return err
	}
	dstAddress, err := dst.GetAddress()
	if err != nil {
		logger.Error(fmt.Sprintf("- [%s]@{%d} - error getting address",
			dst.ChainID(), dstCtx.Height().GetRevisionHeight()), zap.Error(err))
		return err
	}

	if len(sp.Src) > 0 {
		hs, err := sh.SetupHeadersForUpdate(src, dst)
		if err != nil {
			logger.Error(fmt.Sprintf("- [%s]@{%d} - error setting up headers for update",
				src.ChainID(), srcCtx.Height().GetRevisionHeight()), zap.Error(err))
			return err
		}
		if len(hs) > 0 {
			msgs.Dst = dst.Path().UpdateClients(hs, dstAddress)
		}
	}

	if len(sp.Dst) > 0 {
		hs, err := sh.SetupHeadersForUpdate(dst, src)
		if err != nil {
			logger.Error(fmt.Sprintf("- [%s]@{%d} - error setting up headers for update",
				dst.ChainID(), dstCtx.Height().GetRevisionHeight()), zap.Error(err))
			return err
		}
		if len(hs) > 0 {
			msgs.Src = src.Path().UpdateClients(hs, srcAddress)
		}
	}

	packetsForDst, err := collectPackets(srcCtx, src, sp.Src, dstAddress)
	if err != nil {
		logger.Error(fmt.Sprintf("- [%s]@{%d} - error collecting packets",
			src.ChainID(), srcCtx.Height().GetRevisionHeight()), zap.Error(err))
		return err
	}
	packetsForSrc, err := collectPackets(dstCtx, dst, sp.Dst, srcAddress)
	if err != nil {
		logger.Error(fmt.Sprintf("- [%s]@{%d} - error collecting packets",
			dst.ChainID(), dstCtx.Height().GetRevisionHeight()), zap.Error(err))
		return err
	}

	if len(packetsForDst) == 0 && len(packetsForSrc) == 0 {
		logger.Info(fmt.Sprintf("- No packets to relay between [%s]port{%s} and [%s]port{%s}",
			src.ChainID(), src.Path().PortID, dst.ChainID(), dst.Path().PortID))
		return nil
	}

	msgs.Dst = append(msgs.Dst, packetsForDst...)
	msgs.Src = append(msgs.Src, packetsForSrc...)

	// send messages to their respective chains
	if msgs.Send(src, dst); msgs.Success() {
		if num := len(packetsForDst); num > 0 {
			logPacketsRelayed(dst, src, num)
		}
		if num := len(packetsForSrc); num > 0 {
			logPacketsRelayed(src, dst, num)
		}
	}

	return nil
}

func (st NaiveStrategy) UnrelayedAcknowledgements(src, dst *ProvableChain, sh SyncHeaders) (*RelaySequences, error) {
	logger := logger.ZapLogger()
	defer logger.Sync()
	var (
		eg           = new(errgroup.Group)
		srcPacketSeq = []uint64{}
		dstPacketSeq = []uint64{}
		err          error
		rs           = &RelaySequences{Src: []uint64{}, Dst: []uint64{}}
	)

	srcCtx := sh.GetQueryContext(src.ChainID())
	dstCtx := sh.GetQueryContext(dst.ChainID())

	eg.Go(func() error {
		var res *chantypes.QueryPacketAcknowledgementsResponse
		if err = retry.Do(func() error {
			// Query the packet commitment
			res, err = src.QueryPacketAcknowledgementCommitments(srcCtx, 0, 1000)
			switch {
			case err != nil:
				logger.Error(fmt.Sprintf("- [%s]@{%d} - error querying packet commitments",
					src.ChainID(), srcCtx.Height().GetRevisionHeight()), zap.Error(err))
				return err
			case res == nil:
				logger.Error(fmt.Sprintf("- [%s]@{%d} - nil packet commitments",
					src.ChainID(), srcCtx.Height().GetRevisionHeight()))
				return fmt.Errorf("no error on QueryPacketUnrelayedAcknowledgements for %s, however response is nil", src.ChainID())
			default:
				return nil
			}
		}, rtyAtt, rtyDel, rtyErr, retry.OnRetry(func(n uint, err error) {
			logger.Info(fmt.Sprintf("- [%s]@{%d} - try(%d/%d) query packet commitments: %s",
				src.ChainID(), srcCtx.Height().GetRevisionHeight(), n+1, rtyAttNum, err))
			sh.Updates(src, dst)
		})); err != nil {
			logger.Error(fmt.Sprintf("- [%s]@{%d} - max retry exceeded querying packet commitments",
				src.ChainID(), srcCtx.Height().GetRevisionHeight()), zap.Error(err))
			return err
		}
		for _, pc := range res.Acknowledgements {
			srcPacketSeq = append(srcPacketSeq, pc.Sequence)
		}
		return nil
	})

	eg.Go(func() error {
		var res *chantypes.QueryPacketAcknowledgementsResponse
		if err = retry.Do(func() error {
			res, err = dst.QueryPacketAcknowledgementCommitments(dstCtx, 0, 1000)
			switch {
			case err != nil:
				logger.Error(fmt.Sprintf("- [%s]@{%d} - error querying packet commitments",
					dst.ChainID(), dstCtx.Height().GetRevisionHeight()), zap.Error(err))
				return err
			case res == nil:
				logger.Error(fmt.Sprintf("- [%s]@{%d} - nil packet commitments",
					dst.ChainID(), dstCtx.Height().GetRevisionHeight()))
				return fmt.Errorf("no error on QueryPacketUnrelayedAcknowledgements for %s, however response is nil", dst.ChainID())
			default:
				return nil
			}
		}, rtyAtt, rtyDel, rtyErr, retry.OnRetry(func(n uint, err error) {
			logger.Info(fmt.Sprintf("- [%s]@{%d} - try(%d/%d) query packet commitments",
				dst.ChainID(), dstCtx.Height().GetRevisionHeight(), n+1, rtyAttNum), zap.Error(err))
			sh.Updates(src, dst)
		})); err != nil {
			return err
		}
		for _, pc := range res.Acknowledgements {
			dstPacketSeq = append(dstPacketSeq, pc.Sequence)
		}
		return nil
	})

	if err := eg.Wait(); err != nil {
		logger.Error(fmt.Sprintf("- [%s]@{%d} - error querying packet commitments",
			src.ChainID(), srcCtx.Height().GetRevisionHeight()), zap.Error(err))
		return nil, err
	}

	eg.Go(func() error {
		// Query all packets sent by src that have been received by dst
		src, err := dst.QueryUnrecievedAcknowledgements(dstCtx, srcPacketSeq)
		// return err
		if err != nil {
			logger.Error(fmt.Sprintf("- [%s]@{%d} - error querying unrelayed acknowledgements",
				dst.ChainID(), dstCtx.Height().GetRevisionHeight()), zap.Error(err))
			return err
		} else if src != nil {
			rs.Src = src
		}
		return nil
	})

	eg.Go(func() error {
		// Query all packets sent by dst that have been received by src
		dst, err := src.QueryUnrecievedAcknowledgements(srcCtx, dstPacketSeq)
		if err != nil {
			logger.Error(fmt.Sprintf("- [%s]@{%d} - error querying unrelayed acknowledgements",
				src.ChainID(), srcCtx.Height().GetRevisionHeight()), zap.Error(err))
			return err
		} else if dst != nil {
			rs.Dst = dst
		}
		return nil
	})

	if err := eg.Wait(); err != nil {
		logger.Error(fmt.Sprintf("- [%s]@{%d} - error querying unrelayed acknowledgements",
			src.ChainID(), srcCtx.Height().GetRevisionHeight()), zap.Error(err))
		return nil, err
	}

	return rs, nil
}

// TODO add packet-timeout support
func collectPackets(ctx QueryContext, chain *ProvableChain, seqs []uint64, signer sdk.AccAddress) ([]sdk.Msg, error) {
	var msgs []sdk.Msg
	for _, seq := range seqs {
		p, err := chain.QueryPacket(ctx, seq)
		if err != nil {
			log.Println("failed to QueryPacket:", ctx.Height(), seq, err)
			return nil, err
		}
		res, err := chain.QueryPacketCommitmentWithProof(ctx, seq)
		if err != nil {
			log.Println("failed to QueryPacketCommitment:", ctx.Height(), seq, err)
			return nil, err
		}
		msg := chantypes.NewMsgRecvPacket(*p, res.Proof, res.ProofHeight, signer.String())
		msgs = append(msgs, msg)
	}
	return msgs, nil
}

func logPacketsRelayed(src, dst Chain, num int) {
	logger := logger.ZapLogger()
	defer logger.Sync()
	logger.Info(fmt.Sprintf("★ Relayed %d packets: [%s]port{%s}->[%s]port{%s}",
		num, dst.ChainID(), dst.Path().PortID, src.ChainID(), src.Path().PortID))
}

func (st NaiveStrategy) RelayAcknowledgements(src, dst *ProvableChain, sp *RelaySequences, sh SyncHeaders) error {
	logger := logger.ZapLogger()
	defer logger.Sync()
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
		logger.Error(fmt.Sprintf("- [%s]@{%d} - error getting address",
			src.ChainID(), srcCtx.Height().GetRevisionHeight()), zap.Error(err))
		return err
	}
	dstAddress, err := dst.GetAddress()
	if err != nil {
		logger.Error(fmt.Sprintf("- [%s]@{%d} - error getting address",
			dst.ChainID(), dstCtx.Height().GetRevisionHeight()), zap.Error(err))
		return err
	}

	if len(sp.Src) > 0 {
		hs, err := sh.SetupHeadersForUpdate(src, dst)
		if err != nil {
			logger.Error(fmt.Sprintf("- [%s]@{%d} - error setting up headers",
				src.ChainID(), srcCtx.Height().GetRevisionHeight()), zap.Error(err))
			return err
		}
		if len(hs) > 0 {
			msgs.Dst = dst.Path().UpdateClients(hs, dstAddress)
		}
	}

	if len(sp.Dst) > 0 {
		hs, err := sh.SetupHeadersForUpdate(dst, src)
		if err != nil {
			logger.Error(fmt.Sprintf("- [%s]@{%d} - error setting up headers",
				dst.ChainID(), dstCtx.Height().GetRevisionHeight()), zap.Error(err))
			return err
		}
		if len(hs) > 0 {
			msgs.Src = src.Path().UpdateClients(hs, srcAddress)
		}
	}

	acksForDst, err := collectAcks(dstCtx, srcCtx, dst, src, sp.Src, dstAddress)
	if err != nil {
		logger.Error(fmt.Sprintf("- [%s]@{%d} - error querying unrelayed acknowledgements",
			dst.ChainID(), dstCtx.Height().GetRevisionHeight()), zap.Error(err))
		return err
	}
	acksForSrc, err := collectAcks(srcCtx, dstCtx, src, dst, sp.Dst, srcAddress)
	if err != nil {
		logger.Error(fmt.Sprintf("- [%s]@{%d} - error querying unrelayed acknowledgements",
			src.ChainID(), srcCtx.Height().GetRevisionHeight()), zap.Error(err))
		return err
	}

	if len(acksForDst) == 0 && len(acksForSrc) == 0 {
		logger.Info(fmt.Sprintf("- No acknowledgements to relay between [%s]port{%s} and [%s]port{%s}",
			src.ChainID(), src.Path().PortID, dst.ChainID(), dst.Path().PortID))
		return nil
	}

	msgs.Dst = append(msgs.Dst, acksForDst...)
	msgs.Src = append(msgs.Src, acksForSrc...)

	// send messages to their respective chains
	if msgs.Send(src, dst); msgs.Success() {
		if num := len(acksForDst); num > 0 {
			logPacketsRelayed(dst, src, num)
		}
		if num := len(acksForSrc); num > 0 {
			logPacketsRelayed(src, dst, num)
		}
	}

	return nil
}

func collectAcks(senderCtx, receiverCtx QueryContext, senderChain, receiverChain *ProvableChain, seqs []uint64, signer sdk.AccAddress) ([]sdk.Msg, error) {
	var msgs []sdk.Msg

	for _, seq := range seqs {
		p, err := senderChain.QueryPacket(senderCtx, seq)
		if err != nil {
			return nil, err
		}
		ack, err := receiverChain.QueryPacketAcknowledgement(receiverCtx, seq)
		if err != nil {
			return nil, err
		}
		res, err := receiverChain.QueryPacketAcknowledgementCommitmentWithProof(receiverCtx, seq)
		if err != nil {
			return nil, err
		}

		msg := chantypes.NewMsgAcknowledgement(*p, ack, res.Proof, res.ProofHeight, signer.String())
		msgs = append(msgs, msg)
	}

	return msgs, nil
}
