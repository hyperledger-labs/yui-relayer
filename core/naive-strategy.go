package core

import (
	"context"
	"errors"
	"fmt"
	"log"

	retry "github.com/avast/retry-go"
	sdk "github.com/cosmos/cosmos-sdk/types"
	chantypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	"github.com/hyperledger-labs/yui-relayer/logger"
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
	zapLogger := logger.GetLogger()
	defer zapLogger.Zap.Sync()
	if err := src.SetupForRelay(ctx); err != nil {
		naiveErrorwChannel(
			zapLogger,
			"failed to setup for src",
			src, dst,
			err,
		)
		return err
	}
	if err := dst.SetupForRelay(ctx); err != nil {
		naiveErrorwChannel(
			zapLogger,
			"failed to setup for dst",
			src, dst,
			err,
		)
		return err
	}
	return nil
}

func (st NaiveStrategy) UnrelayedSequences(src, dst *ProvableChain, sh SyncHeaders) (*RelaySequences, error) {
	zapLogger := logger.GetLogger()
	defer zapLogger.Zap.Sync()
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
				naiveErrorwChannel(
					zapLogger,
					"error querying packet commitments",
					src, dst,
					err,
				)
				return err
			case res == nil:
				naiveErrorwChannel(
					zapLogger,
					"nil packet commitments",
					src, dst,
					errors.New("however response is nil"),
				)
				return fmt.Errorf("no error on QueryPacketCommitments for %s, however response is nil", src.ChainID())
			default:
				return nil
			}
		}, rtyAtt, rtyDel, rtyErr, retry.OnRetry(func(n uint, err error) {
			zapLogger.Zap.Info(
				fmt.Sprintf("- [%s]@{%d} - try(%d/%d) query packet commitments: %s", src.ChainID(), srcCtx.Height().GetRevisionHeight(), n+1, rtyAttNum, err),
			)
		})); err != nil {
			naiveErrorwChannel(
				zapLogger,
				"max retry exceeded querying packet commitments",
				src, dst,
				err,
			)
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
				naiveErrorwChannel(
					zapLogger,
					"error querying packet commitments",
					src, dst,
					err,
				)
				return err
			case res == nil:
				naiveErrorwChannel(
					zapLogger,
					"nil packet commitments",
					src, dst,
					errors.New("however response is nil"),
				)
				return fmt.Errorf("no error on QueryPacketCommitments for %s, however response is nil", dst.ChainID())
			default:
				return nil
			}
		}, rtyAtt, rtyDel, rtyErr, retry.OnRetry(func(n uint, err error) {
			naiveInfowChannel(
				zapLogger,
				"query package commitments",
				src, dst,
				fmt.Sprintf("- [%s]@{%d} - try(%d/%d) query packet commitments: %s", dst.ChainID(), dstCtx.Height().GetRevisionHeight(), n+1, rtyAttNum, err),
			)
		})); err != nil {
			naiveErrorwChannel(
				zapLogger,
				"max retry exceeded querying packet commitments",
				src, dst,
				err,
			)
			return err
		}
		for _, pc := range res.Commitments {
			dstPacketSeq = append(dstPacketSeq, pc.Sequence)
		}
		return nil
	})

	if err := eg.Wait(); err != nil {
		naiveErrorwChannel(
			zapLogger,
			"error querying packet commitments",
			src, dst,
			err,
		)
		return nil, err
	}

	eg.Go(func() error {
		// Query all packets sent by src that have been received by dst
		srcPacket, err := dst.QueryUnrecievedPackets(dstCtx, srcPacketSeq)
		if err != nil {
			naiveErrorwChannel(
				zapLogger,
				"error querying unrelayed packets",
				src, dst,
				err,
			)
			return err
		} else if srcPacket != nil {
			rs.Src = srcPacket
		}
		return nil
	})

	eg.Go(func() error {
		// Query all packets sent by dst that have been received by src
		dstPacket, err := src.QueryUnrecievedPackets(srcCtx, dstPacketSeq)
		if err != nil {
			naiveErrorwChannel(
				zapLogger,
				"error querying unrelayed packets",
				src, dst,
				err,
			)
			return err
		} else if dstPacket != nil {
			rs.Dst = dstPacket
		}
		return nil
	})

	if err := eg.Wait(); err != nil {
		naiveErrorwChannel(
			zapLogger,
			"error querying unrelayed packets",
			src, dst,
			err,
		)
		return nil, err
	}

	return rs, nil
}

func (st NaiveStrategy) RelayPackets(src, dst *ProvableChain, sp *RelaySequences, sh SyncHeaders) error {
	zapLogger := logger.GetLogger()
	defer zapLogger.Zap.Sync()
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
		naiveErrorwChannel(
			zapLogger,
			"error getting address",
			src, dst,
			err,
		)
		return err
	}
	dstAddress, err := dst.GetAddress()
	if err != nil {
		naiveErrorwChannel(
			zapLogger,
			"error getting address",
			src, dst,
			err,
		)
		return err
	}

	if len(sp.Src) > 0 {
		hs, err := sh.SetupHeadersForUpdate(src, dst)
		if err != nil {
			naiveErrorwChannel(
				zapLogger,
				"error setting up headers for update",
				src, dst,
				err,
			)
			return err
		}
		if len(hs) > 0 {
			msgs.Dst = dst.Path().UpdateClients(hs, dstAddress)
		}
	}

	if len(sp.Dst) > 0 {
		hs, err := sh.SetupHeadersForUpdate(dst, src)
		if err != nil {
			naiveErrorwChannel(
				zapLogger,
				"error setting up headers for update",
				src, dst,
				err,
			)
			return err
		}
		if len(hs) > 0 {
			msgs.Src = src.Path().UpdateClients(hs, srcAddress)
		}
	}

	packetsForDst, err := collectPackets(srcCtx, src, sp.Src, dstAddress)
	if err != nil {
		naiveErrorwChannel(
			zapLogger,
			"error collecting packets",
			src, dst,
			err,
		)
		return err
	}
	packetsForSrc, err := collectPackets(dstCtx, dst, sp.Dst, srcAddress)
	if err != nil {
		naiveErrorwChannel(
			zapLogger,
			"error collecting packets",
			src, dst,
			err,
		)
		return err
	}

	if len(packetsForDst) == 0 && len(packetsForSrc) == 0 {
		naiveInfowChannel(
			zapLogger,
			"Nopackates to relay",
			src, dst,
			"",
		)
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
	zapLogger := logger.GetLogger()
	defer zapLogger.Zap.Sync()
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
				naiveErrorwChannel(
					zapLogger,
					"error querying packet acknowledgements",
					src, dst,
					err,
				)
				return err
			case res == nil:
				naiveErrorwChannel(
					zapLogger,
					"nil packet commitments",
					src, dst,
					errors.New("nil packet commitments"),
				)
				return fmt.Errorf("no error on QueryPacketUnrelayedAcknowledgements for %s, however response is nil", src.ChainID())
			default:
				return nil
			}
		}, rtyAtt, rtyDel, rtyErr, retry.OnRetry(func(n uint, err error) {
			naiveInfowChannel(
				zapLogger,
				"query packet acknowledgements",
				src, dst,
				fmt.Sprintf("- [%s]@{%d} - try(%d/%d) query packet acknowledgements [Error: %s]", src.ChainID(), srcCtx.Height().GetRevisionHeight(), n+1, rtyAttNum, err.Error()),
			)
			sh.Updates(src, dst)
		})); err != nil {
			naiveErrorwChannel(
				zapLogger,
				"max retry exceeded querying packet acknowledgements",
				src, dst,
				err,
			)
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
				naiveErrorwChannel(
					zapLogger,
					"error querying packet acknowledgements",
					src, dst,
					err,
				)
				return err
			case res == nil:
				naiveErrorwChannel(
					zapLogger,
					"nil packet acknowledgements acknowledgements",
					src, dst,
					errors.New("nil packet acknowledgements"),
				)
				return fmt.Errorf("no error on QueryPacketUnrelayedAcknowledgements for %s, however response is nil", dst.ChainID())
			default:
				return nil
			}
		}, rtyAtt, rtyDel, rtyErr, retry.OnRetry(func(n uint, err error) {
			naiveInfowChannel(
				zapLogger,
				"query packet acknowledgements",
				src, dst,
				fmt.Sprintf("- [%s]@{%d} - try(%d/%d) query packet acknowledgements [Error: %s]", dst.ChainID(), dstCtx.Height().GetRevisionHeight(), n+1, rtyAttNum, err.Error()),
			)
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
		naiveErrorwChannel(
			zapLogger,
			"error querying packet commitments",
			src, dst,
			err,
		)
		return nil, err
	}

	eg.Go(func() error {
		// Query all packets sent by src that have been received by dst
		srcPacket, err := dst.QueryUnrecievedAcknowledgements(dstCtx, srcPacketSeq)
		// return err
		if err != nil {
			naiveErrorwChannel(
				zapLogger,
				"error querying unrelayed acknowledgements",
				src, dst,
				err,
			)
			return err
		} else if srcPacket != nil {
			rs.Src = srcPacket
		}
		return nil
	})

	eg.Go(func() error {
		// Query all packets sent by dst that have been received by src
		dstPacket, err := src.QueryUnrecievedAcknowledgements(srcCtx, dstPacketSeq)
		if err != nil {
			naiveErrorwChannel(
				zapLogger,
				"error querying unrelayed acknowledgements",
				src, dst,
				err,
			)
			return err
		} else if dstPacket != nil {
			rs.Dst = dstPacket
		}
		return nil
	})

	if err := eg.Wait(); err != nil {
		naiveErrorwChannel(
			zapLogger,
			"error querying unrelayed acknowledgements",
			src, dst,
			err,
		)
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
	zapLogger := logger.GetLogger()
	defer zapLogger.Zap.Sync()
	zapLogger.InfowChannel(
		fmt.Sprintf("★ Relayed %d packets", num),
		src.ChainID(), src.Path().ChannelID, src.Path().PortID,
		dst.ChainID(), dst.Path().ChannelID, dst.Path().PortID,
		"",
	)
}

func (st NaiveStrategy) RelayAcknowledgements(src, dst *ProvableChain, sp *RelaySequences, sh SyncHeaders) error {
	zapLogger := logger.GetLogger()
	defer zapLogger.Zap.Sync()
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
		naiveErrorwChannel(
			zapLogger,
			"error getting address",
			src, dst,
			err,
		)
		return err
	}
	dstAddress, err := dst.GetAddress()
	if err != nil {
		naiveErrorwChannel(
			zapLogger,
			"error getting address",
			src, dst,
			err,
		)
		return err
	}

	if len(sp.Src) > 0 {
		hs, err := sh.SetupHeadersForUpdate(src, dst)
		if err != nil {
			naiveErrorwChannel(
				zapLogger,
				"error setting up headers",
				src, dst,
				err,
			)
			return err
		}
		if len(hs) > 0 {
			msgs.Dst = dst.Path().UpdateClients(hs, dstAddress)
		}
	}

	if len(sp.Dst) > 0 {
		hs, err := sh.SetupHeadersForUpdate(dst, src)
		if err != nil {
			naiveErrorwChannel(
				zapLogger,
				"error setting up headers",
				src, dst,
				err,
			)
			return err
		}
		if len(hs) > 0 {
			msgs.Src = src.Path().UpdateClients(hs, srcAddress)
		}
	}

	acksForDst, err := collectAcks(dstCtx, srcCtx, dst, src, sp.Src, dstAddress)
	if err != nil {
		naiveErrorwChannel(
			zapLogger,
			"error querying unrelayed acknowledgements",
			src, dst,
			err,
		)
		return err
	}
	acksForSrc, err := collectAcks(srcCtx, dstCtx, src, dst, sp.Dst, srcAddress)
	if err != nil {
		naiveErrorwChannel(
			zapLogger,
			"error querying unrelayed acknowledgements",
			src, dst,
			err,
		)
		return err
	}

	if len(acksForDst) == 0 && len(acksForSrc) == 0 {
		naiveInfowChannel(
			zapLogger,
			"no acknowledgements to relay",
			src, dst,
			"",
		)
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

func naiveErrorwChannel(zapLogger *logger.ZapLogger, msg string, src, dst *ProvableChain, err error) {
	zapLogger.ErrorwChannel(msg,
		src.ChainID(), src.Path().ChannelID, src.Path().PortID,
		dst.ChainID(), dst.Path().ChannelID, dst.Path().PortID,
		err,
		"core.naive-strategy",
	)
}

func naiveInfowChannel(zapLogger *logger.ZapLogger, msg string, src, dst *ProvableChain, info string) {
	zapLogger.InfowChannel(
		msg,
		src.ChainID(), src.Path().ChannelID, src.Path().PortID,
		dst.ChainID(), dst.Path().ChannelID, dst.Path().PortID,
		info,
	)
}
