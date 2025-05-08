package core

import (
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	chantypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
)

// PacketInfo represents the packet information that is acquired from a SendPacket event or
// a pair of RecvPacket/WriteAcknowledgement events. In the former case, the `Acknowledgement`
// field becomes nil. In the latter case, `EventHeight` represents the height in which the
// underlying `RecvPacket` event occurs.
type PacketInfo struct {
	chantypes.Packet
	Acknowledgement []byte             `json:"acknowledgement"`
	EventHeight     clienttypes.Height `json:"event_height"`
	
	// TimedOut indicates whether the packet has timed out. This is determined based on
	// the absence of a corresponding acknowledgment or receipt within the expected timeframe.
	TimedOut        bool               `json:"timed_out"`
}

// PacketInfoList represents a list of PacketInfo that is sorted in the order in which
// underlying events (SendPacket and RecvPacket) occur.
type PacketInfoList []*PacketInfo

func (ps PacketInfoList) ExtractSequenceList() []uint64 {
	var seqs []uint64
	for _, p := range ps {
		seqs = append(seqs, p.Sequence)
	}
	return seqs
}

func (ps PacketInfoList) Subtract(seqs []uint64) PacketInfoList {
	var ret PacketInfoList
out:
	for _, p := range ps {
		for _, seq := range seqs {
			if p.Sequence == seq {
				continue out
			}
		}
		ret = append(ret, p)
	}
	return ret
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
