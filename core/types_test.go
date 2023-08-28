package core_test

import (
	"math/rand"
	"slices"
	"testing"

	chantypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	"github.com/hyperledger-labs/yui-relayer/core"
)

func makePacketInfoList(seqs ...uint64) core.PacketInfoList {
	var packets core.PacketInfoList
	for _, seq := range seqs {
		packets = append(packets, &core.PacketInfo{Packet: chantypes.Packet{Sequence: seq}})
	}
	return packets
}

func TestPacketInfoList(t *testing.T) {
	var expectedSeqs []uint64
	for i := 0; i < 10; i++ {
		expectedSeqs = append(expectedSeqs, rand.Uint64())
	}
	seqs := makePacketInfoList(expectedSeqs...).ExtractSequenceList()
	if len(seqs) != 10 || !slices.Equal(seqs, expectedSeqs) {
		t.Errorf("ExtractSequenceList returns an unexpected result: actual=%v, expected=%v", seqs, expectedSeqs)
	}

	packets := makePacketInfoList(3, 4, 5, 6, 7).Filter([]uint64{5, 6, 7, 8, 9})
	expectedPackets := makePacketInfoList(5, 6, 7)
	if !slices.Equal(packets.ExtractSequenceList(), expectedPackets.ExtractSequenceList()) {
		t.Errorf("Filter returns an unexpected result: actual=%v, expected=%v", packets, expectedPackets)
	}

	packets = makePacketInfoList(7, 6, 5, 4, 3).Filter([]uint64{5, 6, 7, 8, 9})
	expectedPackets = makePacketInfoList(7, 6, 5)
	if !slices.Equal(packets.ExtractSequenceList(), expectedPackets.ExtractSequenceList()) {
		t.Errorf("Filter returns an unexpected result: actual=%v, expected=%v", packets, expectedPackets)
	}

	packets = makePacketInfoList(3, 4, 5, 6, 7).Subtract([]uint64{5, 6, 7, 8, 9})
	expectedPackets = makePacketInfoList(3, 4)
	if !slices.Equal(packets.ExtractSequenceList(), expectedPackets.ExtractSequenceList()) {
		t.Errorf("Subtract returns an unexpected result: actual=%v, expected=%v", packets, expectedPackets)
	}

	packets = makePacketInfoList(7, 6, 5, 4, 3).Subtract([]uint64{5, 6, 7, 8, 9})
	expectedPackets = makePacketInfoList(4, 3)
	if !slices.Equal(packets.ExtractSequenceList(), expectedPackets.ExtractSequenceList()) {
		t.Errorf("Subtract returns an unexpected result: actual=%v, expected=%v", packets, expectedPackets)
	}
}
