package core

import (
	"fmt"
	"log"

	retry "github.com/avast/retry-go"
	sdk "github.com/cosmos/cosmos-sdk/types"
	chantypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
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

func (st NaiveStrategy) UnrelayedSequences(src, dst *ProvableChain, sh SyncHeadersI) (*RelaySequences, error) {
	var (
		eg           = new(errgroup.Group)
		srcPacketSeq = []uint64{}
		dstPacketSeq = []uint64{}
		err          error
		rs           = &RelaySequences{Src: []uint64{}, Dst: []uint64{}}
	)

	eg.Go(func() error {
		var res *chantypes.QueryPacketCommitmentsResponse
		if err = retry.Do(func() error {
			// Query the packet commitment
			res, err = src.QueryPacketCommitments(0, 1000, sh.GetQueryableHeight(src.ChainID()))
			switch {
			case err != nil:
				return err
			case res == nil:
				return fmt.Errorf("No error on QueryPacketCommitments for %s, however response is nil", src.ChainID())
			default:
				return nil
			}
		}, rtyAtt, rtyDel, rtyErr, retry.OnRetry(func(n uint, err error) {
			log.Println(fmt.Sprintf("- [%s]@{%d} - try(%d/%d) query packet commitments: %s", src.ChainID(), sh.GetQueryableHeight(src.ChainID()), n+1, rtyAttNum, err))
		})); err != nil {
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
			res, err = dst.QueryPacketCommitments(0, 1000, sh.GetQueryableHeight(dst.ChainID()))
			switch {
			case err != nil:
				return err
			case res == nil:
				return fmt.Errorf("No error on QueryPacketCommitments for %s, however response is nil", dst.ChainID())
			default:
				return nil
			}
		}, rtyAtt, rtyDel, rtyErr, retry.OnRetry(func(n uint, err error) {
			log.Println(fmt.Sprintf("- [%s]@{%d} - try(%d/%d) query packet commitments: %s", dst.ChainID(), sh.GetQueryableHeight(dst.ChainID()), n+1, rtyAttNum, err))
		})); err != nil {
			return err
		}
		for _, pc := range res.Commitments {
			dstPacketSeq = append(dstPacketSeq, pc.Sequence)
		}
		return nil
	})

	if err := eg.Wait(); err != nil {
		return nil, err
	}

	eg.Go(func() error {
		// Query all packets sent by src that have been received by dst
		src, err := dst.QueryUnrecievedPackets(sh.GetQueryableHeight(dst.ChainID()), srcPacketSeq)
		if err != nil {
			return err
		} else if src != nil {
			rs.Src = src
		}
		return nil
	})

	eg.Go(func() error {
		// Query all packets sent by dst that have been received by src
		dst, err := src.QueryUnrecievedPackets(sh.GetQueryableHeight(src.ChainID()), dstPacketSeq)
		if err != nil {
			return err
		} else if dst != nil {
			rs.Dst = dst
		}
		return nil
	})

	if err := eg.Wait(); err != nil {
		return nil, err
	}

	return rs, nil
}

func (st NaiveStrategy) RelayPackets(src, dst *ProvableChain, sp *RelaySequences, sh SyncHeadersI) error {
	// set the maximum relay transaction constraints
	msgs := &RelayMsgs{
		Src:          []sdk.Msg{},
		Dst:          []sdk.Msg{},
		MaxTxSize:    st.MaxTxSize,
		MaxMsgLength: st.MaxMsgLength,
	}
	addr, err := dst.GetAddress()
	if err != nil {
		return err
	}
	msgs.Dst, err = relayPackets(src, sp.Src, sh, addr)
	if err != nil {
		return err
	}
	addr, err = src.GetAddress()
	if err != nil {
		return err
	}
	msgs.Src, err = relayPackets(dst, sp.Dst, sh, addr)
	if err != nil {
		return err
	}
	if !msgs.Ready() {
		log.Println(fmt.Sprintf("- No packets to relay between [%s]port{%s} and [%s]port{%s}",
			src.ChainID(), src.Path().PortID, dst.ChainID(), dst.Path().PortID))
		return nil
	}

	// Prepend non-empty msg lists with UpdateClient
	if len(msgs.Dst) != 0 {
		// Sending an update from src to dst
		h, err := sh.GetHeader(src, dst)
		if err != nil {
			return err
		}
		addr, err := dst.GetAddress()
		if err != nil {
			return err
		}
		if h != nil {
			msgs.Dst = append([]sdk.Msg{dst.Path().UpdateClient(h, addr)}, msgs.Dst...)
		}
	}

	if len(msgs.Src) != 0 {
		h, err := sh.GetHeader(dst, src)
		if err != nil {
			return err
		}
		addr, err := src.GetAddress()
		if err != nil {
			return err
		}
		if h != nil {
			msgs.Src = append([]sdk.Msg{src.Path().UpdateClient(h, addr)}, msgs.Src...)
		}
	}

	// send messages to their respective chains
	if msgs.Send(src, dst); msgs.Success() {
		if len(msgs.Dst) > 1 {
			logPacketsRelayed(dst, src, len(msgs.Dst)-1)
		}
		if len(msgs.Src) > 1 {
			logPacketsRelayed(src, dst, len(msgs.Src)-1)
		}
	}

	return nil
}

func (st NaiveStrategy) UnrelayedAcknowledgements(src, dst *ProvableChain, sh SyncHeadersI) (*RelaySequences, error) {
	var (
		eg           = new(errgroup.Group)
		srcPacketSeq = []uint64{}
		dstPacketSeq = []uint64{}
		err          error
		rs           = &RelaySequences{Src: []uint64{}, Dst: []uint64{}}
	)

	eg.Go(func() error {
		var res *chantypes.QueryPacketAcknowledgementsResponse
		if err = retry.Do(func() error {
			// Query the packet commitment
			res, err = src.QueryPacketAcknowledgementCommitments(0, 1000, sh.GetQueryableHeight(src.ChainID()))
			switch {
			case err != nil:
				return err
			case res == nil:
				return fmt.Errorf("No error on QueryPacketUnrelayedAcknowledgements for %s, however response is nil", src.ChainID())
			default:
				return nil
			}
		}, rtyAtt, rtyDel, rtyErr, retry.OnRetry(func(n uint, err error) {
			log.Println((fmt.Sprintf("- [%s]@{%d} - try(%d/%d) query packet acknowledgements: %s", src.ChainID(), sh.GetQueryableHeight(src.ChainID()), n+1, rtyAttNum, err)))
			sh.Updates(src, dst)
		})); err != nil {
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
			res, err = dst.QueryPacketAcknowledgementCommitments(0, 1000, sh.GetQueryableHeight(dst.ChainID()))
			switch {
			case err != nil:
				return err
			case res == nil:
				return fmt.Errorf("No error on QueryPacketUnrelayedAcknowledgements for %s, however response is nil", dst.ChainID())
			default:
				return nil
			}
		}, rtyAtt, rtyDel, rtyErr, retry.OnRetry(func(n uint, err error) {
			log.Println((fmt.Sprintf("- [%s]@{%d} - try(%d/%d) query packet acknowledgements: %s", dst.ChainID(), sh.GetQueryableHeight(dst.ChainID()), n+1, rtyAttNum, err)))
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
		return nil, err
	}

	eg.Go(func() error {
		// Query all packets sent by src that have been received by dst
		src, err := dst.QueryUnrecievedAcknowledgements(sh.GetQueryableHeight(dst.ChainID()), srcPacketSeq)
		// return err
		if err != nil {
			return err
		} else if src != nil {
			rs.Src = src
		}
		return nil
	})

	eg.Go(func() error {
		// Query all packets sent by dst that have been received by src
		dst, err := src.QueryUnrecievedAcknowledgements(sh.GetQueryableHeight(src.ChainID()), dstPacketSeq)
		if err != nil {
			return err
		} else if dst != nil {
			rs.Dst = dst
		}
		return nil
	})

	if err := eg.Wait(); err != nil {
		return nil, err
	}

	return rs, nil
}

// TODO add packet-timeout support
func relayPackets(chain *ProvableChain, seqs []uint64, sh SyncHeadersI, sender sdk.AccAddress) ([]sdk.Msg, error) {
	var msgs []sdk.Msg
	for _, seq := range seqs {
		p, err := chain.QueryPacket(int64(sh.GetQueryableHeight(chain.ChainID())), seq)
		if err != nil {
			log.Println("failed to QueryPacket:", int64(sh.GetQueryableHeight(chain.ChainID())), seq, err)
			return nil, err
		}
		provableHeight := sh.GetProvableHeight(chain.ChainID())
		res, err := chain.QueryPacketCommitmentWithProof(provableHeight, seq)
		if err != nil {
			log.Println("failed to QueryPacketCommitment:", provableHeight, seq, err)
			return nil, err
		}
		msg := chantypes.NewMsgRecvPacket(*p, res.Proof, res.ProofHeight, sender.String())
		msgs = append(msgs, msg)
	}
	return msgs, nil
}

func logPacketsRelayed(src, dst ChainI, num int) {
	log.Println(fmt.Sprintf("★ Relayed %d packets: [%s]port{%s}->[%s]port{%s}",
		num, dst.ChainID(), dst.Path().PortID, src.ChainID(), src.Path().PortID))
}

func (st NaiveStrategy) RelayAcknowledgements(src, dst *ProvableChain, sp *RelaySequences, sh SyncHeadersI) error {
	// set the maximum relay transaction constraints
	msgs := &RelayMsgs{
		Src:          []sdk.Msg{},
		Dst:          []sdk.Msg{},
		MaxTxSize:    st.MaxTxSize,
		MaxMsgLength: st.MaxMsgLength,
	}

	addr, err := dst.GetAddress()
	if err != nil {
		return err
	}
	msgs.Dst, err = relayAcks(src, dst, sp.Src, sh, addr)
	if err != nil {
		return err
	}
	addr, err = src.GetAddress()
	if err != nil {
		return err
	}
	msgs.Src, err = relayAcks(dst, src, sp.Dst, sh, addr)
	if err != nil {
		return err
	}
	if !msgs.Ready() {
		log.Println(fmt.Sprintf("- No acknowledgements to relay between [%s]port{%s} and [%s]port{%s}",
			src.ChainID(), src.Path().PortID, dst.ChainID(), dst.Path().PortID))
		return nil
	}

	// Prepend non-empty msg lists with UpdateClient
	if len(msgs.Dst) != 0 {
		h, err := sh.GetHeader(src, dst)
		if err != nil {
			return err
		}
		addr, err := dst.GetAddress()
		if err != nil {
			return err
		}
		if h != nil {
			msgs.Dst = append([]sdk.Msg{dst.Path().UpdateClient(h, addr)}, msgs.Dst...)
		}
	}

	if len(msgs.Src) != 0 {
		h, err := sh.GetHeader(dst, src)
		if err != nil {
			return err
		}
		addr, err := src.GetAddress()
		if err != nil {
			return err
		}
		if h != nil {
			msgs.Src = append([]sdk.Msg{src.Path().UpdateClient(h, addr)}, msgs.Src...)
		}
	}

	// send messages to their respective chains
	if msgs.Send(src, dst); msgs.Success() {
		if len(msgs.Dst) > 1 {
			logPacketsRelayed(dst, src, len(msgs.Dst)-1)
		}
		if len(msgs.Src) > 1 {
			logPacketsRelayed(src, dst, len(msgs.Src)-1)
		}
	}

	return nil
}

func relayAcks(receiverChain, senderChain *ProvableChain, seqs []uint64, sh SyncHeadersI, sender sdk.AccAddress) ([]sdk.Msg, error) {
	var msgs []sdk.Msg

	for _, seq := range seqs {
		p, err := senderChain.QueryPacket(sh.GetQueryableHeight(senderChain.ChainID()), seq)
		if err != nil {
			return nil, err
		}
		ack, err := receiverChain.QueryPacketAcknowledgement(sh.GetQueryableHeight(receiverChain.ChainID()), seq)
		if err != nil {
			return nil, err
		}
		provableHeight := sh.GetProvableHeight(receiverChain.ChainID())
		res, err := receiverChain.QueryPacketAcknowledgementCommitmentWithProof(provableHeight, seq)
		if err != nil {
			return nil, err
		}

		msg := chantypes.NewMsgAcknowledgement(*p, ack, res.Proof, res.ProofHeight, sender.String())
		msgs = append(msgs, msg)
	}

	return msgs, nil
}
