package core

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
	abci "github.com/tendermint/tendermint/abci/types"
)

func GetPacketsFromEvents(events []abci.Event) ([]channeltypes.Packet, error) {
	var packets []channeltypes.Packet
	for _, ev := range events {
		if ev.Type != channeltypes.EventTypeSendPacket {
			continue
		}
		// NOTE: Attributes of packet are included in one event.
		var (
			packet channeltypes.Packet
			err    error
		)
		for i, attr := range ev.Attributes {
			v := string(attr.Value)
			switch string(attr.Key) {
			case channeltypes.AttributeKeyData:
				// AttributeKeyData key indicates a start of packet attributes
				packet = channeltypes.Packet{}
				packet.Data = []byte(attr.Value)
				err = assertIndex(i, 0)
			case channeltypes.AttributeKeyDataHex:
				var bz []byte
				bz, err = hex.DecodeString(string(attr.Value))
				if err != nil {
					panic(err)
				}
				packet.Data = bz
				err = assertIndex(i, 1)
			case channeltypes.AttributeKeyTimeoutHeight:
				parts := strings.Split(v, "-")
				packet.TimeoutHeight = clienttypes.NewHeight(
					strToUint64(parts[0]),
					strToUint64(parts[1]),
				)
				err = assertIndex(i, 2)
			case channeltypes.AttributeKeyTimeoutTimestamp:
				packet.TimeoutTimestamp = strToUint64(v)
				err = assertIndex(i, 3)
			case channeltypes.AttributeKeySequence:
				packet.Sequence = strToUint64(v)
				err = assertIndex(i, 4)
			case channeltypes.AttributeKeySrcPort:
				packet.SourcePort = v
				err = assertIndex(i, 5)
			case channeltypes.AttributeKeySrcChannel:
				packet.SourceChannel = v
				err = assertIndex(i, 6)
			case channeltypes.AttributeKeyDstPort:
				packet.DestinationPort = v
				err = assertIndex(i, 7)
			case channeltypes.AttributeKeyDstChannel:
				packet.DestinationChannel = v
				err = assertIndex(i, 8)
			}
			if err != nil {
				return nil, err
			}
		}
		if err := packet.ValidateBasic(); err != nil {
			return nil, err
		}
		packets = append(packets, packet)
	}
	return packets, nil
}

func FindPacketFromEventsBySequence(events []abci.Event, seq uint64) (*channeltypes.Packet, error) {
	packets, err := GetPacketsFromEvents(events)
	if err != nil {
		return nil, err
	}
	for _, packet := range packets {
		if packet.Sequence == seq {
			return &packet, nil
		}
	}
	return nil, nil
}

type packetAcknowledgement struct {
	srcPortID    string
	srcChannelID string
	dstPortID    string
	dstChannelID string
	sequence     uint64
	data         []byte
}

func (ack packetAcknowledgement) Data() []byte {
	return ack.data
}

func GetPacketAcknowledgementsFromEvents(events []abci.Event) ([]packetAcknowledgement, error) {
	var acks []packetAcknowledgement
	for _, ev := range events {
		if ev.Type != channeltypes.EventTypeWriteAck {
			continue
		}
		var (
			ack packetAcknowledgement
			err error
		)
		for i, attr := range ev.Attributes {
			v := string(attr.Value)
			switch string(attr.Key) {
			case channeltypes.AttributeKeySequence:
				ack.sequence = strToUint64(v)
				err = assertIndex(i, 4)
			case channeltypes.AttributeKeySrcPort:
				ack.srcPortID = v
				err = assertIndex(i, 5)
			case channeltypes.AttributeKeySrcChannel:
				ack.srcChannelID = v
				err = assertIndex(i, 6)
			case channeltypes.AttributeKeyDstPort:
				ack.dstPortID = v
				err = assertIndex(i, 7)
			case channeltypes.AttributeKeyDstChannel:
				ack.dstChannelID = v
				err = assertIndex(i, 8)
			case channeltypes.AttributeKeyAck:
				ack.data = attr.Value
				err = assertIndex(i, 9)
			}
			if err != nil {
				return nil, err
			}
		}
		acks = append(acks, ack)
	}
	return acks, nil
}

func FindPacketAcknowledgementFromEventsBySequence(events []abci.Event, seq uint64) (*packetAcknowledgement, error) {
	acks, err := GetPacketAcknowledgementsFromEvents(events)
	if err != nil {
		return nil, err
	}
	for _, ack := range acks {
		if ack.sequence == seq {
			return &ack, nil
		}
	}
	return nil, nil
}

func assertIndex(actual, expected int) error {
	if actual == expected {
		return nil
	} else {
		return fmt.Errorf("assertion error: %v != %v", actual, expected)
	}
}

func strToUint64(s string) uint64 {
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		panic(err)
	}
	return uint64(v)
}
