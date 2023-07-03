package core

import (
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	chantypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
)

// PacketInfo represents the packet information that is acquired from a SendPacket event or
// a pair of RecvPacket/WriteAcknowledgement events. In the former case, the `Acknowledgement`
// field becomes nil. In the latter case, `EventHeight` represents the smaller height between
// the ones of the `RecvPacket` event and the `WriteAcknowledgement` event.
type PacketInfo struct {
	chantypes.Packet
	Acknowledgement []byte             `json:"acknowledgement"`
	EventHeight     clienttypes.Height `json:"event_height"`
}

type PacketInfoList []*PacketInfo

func (ps PacketInfoList) ExtractSequenceList() []uint64 {
	var seqs []uint64
	for _, p := range ps {
		seqs = append(seqs, p.Sequence)
	}
	return seqs
}

func (ps PacketInfoList) Filter(seqs []uint64) PacketInfoList {
	var ret PacketInfoList
	for _, p := range ps {
		for _, seq := range seqs {
			if p.Sequence == seq {
				ret = append(ret, p)
				break
			}
		}
	}
	return ret
}

// RelayPackets represents unrelayed packets on src and dst
type RelayPackets struct {
	Src PacketInfoList `json:"src"`
	Dst PacketInfoList `json:"dst"`
}
