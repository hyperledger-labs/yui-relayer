package core

import (
	"context"
	"encoding/hex"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	abci "github.com/cometbft/cometbft/abci/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
)

func GetPacketsFromEvents(events []abci.Event, eventType string) ([]channeltypes.Packet, error) {
	var packets []channeltypes.Packet
	for _, ev := range events {
		if ev.Type != eventType {
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
				bz, err = hex.DecodeString(attr.Value)
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

func FindPacketFromEventsBySequence(events []abci.Event, eventType string, seq uint64) (*channeltypes.Packet, error) {
	packets, err := GetPacketsFromEvents(events, eventType)
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
				ack.data = []byte(attr.Value)
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

// AsChain is a method similar to errors.As and finds the first struct value in the Chain
// field that matches target.
//
// In the following example, AsChain sets a struct value in the Chain field to the chain variable:
//
//	var chain module.Chain
//	if ok := core.AsChain(provableChain, &chain); !ok {
//	        return errors.New("Chain is not a module.Chain")
//	}
func AsChain(v any, target any) bool {
	return as(v, target, "Chain")
}

// AsProver is a method similar to errors.As and finds the first struct value in the Prover
// field that matches target.
//
// In the following example, AsProver sets a struct value in the Prover field to the prover variable:
//
//	var prover module.Prover
//	if ok := core.AsProver(provableChain, &prover); !ok {
//	        return errors.New("Prover is not a module.Prover")
//	}
func AsProver(v any, target any) bool {
	return as(v, target, "Prover")
}

func as(v any, target any, fieldName string) bool {
	targetType := reflect.TypeOf(target).Elem()

	rv := reflect.ValueOf(v)
	for {
		// core.Chain/core.Prover to concrete chain/prover (struct or pointer)
		if rv.Kind() == reflect.Interface {
			rv = rv.Elem()
		}
		// chain/prover pointer to struct
		if rv.Kind() == reflect.Ptr {
			rv = rv.Elem()
		}

		if reflect.TypeOf(rv.Interface()).AssignableTo(targetType) {
			reflect.ValueOf(target).Elem().Set(rv)
			return true
		}

		rv = rv.FieldByName(fieldName)
		if !rv.IsValid() || rv.IsNil() {
			return false
		}
	}
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

func wait(ctx context.Context, d time.Duration) error {
	// NOTE: We can use time.After with Go 1.23 or later
	// cf. https://pkg.go.dev/time@go1.23.0#After
	t := time.NewTimer(d)
	defer t.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

func runUntilComplete(ctx context.Context, interval time.Duration, fn func() (bool, error)) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	if complete, err := fn(); err != nil {
		return err
	} else if complete {
		return nil
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if complete, err := fn(); err != nil {
				return err
			} else if complete {
				return nil
			}
		}
	}
}
