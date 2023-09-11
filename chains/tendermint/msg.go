package tendermint

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	conntypes "github.com/cosmos/ibc-go/v7/modules/core/03-connection/types"
	chantypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
	"github.com/hyperledger-labs/yui-relayer/core"
)

var (
	_ core.MsgID       = (*MsgID)(nil)
	_ core.MsgResult   = (*MsgResult)(nil)
	_ core.MsgEventLog = (*UnparseableEventLog)(nil)
)

type MsgID struct {
	txHash   string
	msgIndex uint32
}

func (i *MsgID) IsMsgID() {}

type MsgResult struct {
	height clienttypes.Height
	status bool
	events []core.MsgEventLog
}

func (r *MsgResult) BlockHeight() clienttypes.Height {
	return r.height
}

func (r *MsgResult) Status() (bool, string) {
	return r.status, ""
}

func (r *MsgResult) Events() []core.MsgEventLog {
	return r.events
}

type UnparseableEventLog struct {
	sdk.StringEvent
}

func (e *UnparseableEventLog) IsMsgEventLog() {}

func parseMsgEventLogs(cdc codec.Codec, logs sdk.ABCIMessageLogs, msgIndex uint32) ([]core.MsgEventLog, error) {
	var msgEventLogs []core.MsgEventLog
	for _, log := range logs {
		if msgIndex == log.MsgIndex {
			for _, ev := range log.Events {
				event, err := parseMsgEventLog(cdc, ev)
				if err != nil {
					return nil, fmt.Errorf("failed to parse msg event log: %v", err)
				}
				msgEventLogs = append(msgEventLogs, event)
			}
		}
	}
	return msgEventLogs, nil
}

func parseMsgEventLog(cdc codec.Codec, ev sdk.StringEvent) (core.MsgEventLog, error) {
	switch ev.Type {
	case clienttypes.EventTypeCreateClient:
		var event core.EventCreateClient
		var err0, err1, err2 error
		event.ClientID, err0 = getAttributeString(ev, clienttypes.AttributeKeyClientID)
		event.ClientType, err1 = getAttributeString(ev, clienttypes.AttributeKeyClientType)
		event.ConsensusHeight, err2 = getAttributeHeight(ev, clienttypes.AttributeKeyConsensusHeight)
		if err := errors.Join(err0, err1, err2); err != nil {
			return nil, err
		}
		return &event, nil
	case clienttypes.EventTypeUpdateClient:
		var event core.EventUpdateClient
		var err0, err1, err2, err3 error
		event.ClientID, err0 = getAttributeString(ev, clienttypes.AttributeKeyClientID)
		event.ClientType, err1 = getAttributeString(ev, clienttypes.AttributeKeyClientType)
		event.ConsensusHeights, err2 = getAttributeHeights(ev, clienttypes.AttributeKeyConsensusHeight)
		event.ClientMessage, err3 = getAttributeClientMessage(cdc, ev, clienttypes.AttributeKeyHeader)
		if err := errors.Join(err0, err1, err2, err3); err != nil {
			return nil, err
		}
		return &event, nil
	case conntypes.EventTypeConnectionOpenInit:
		var event core.EventConnectionOpenInit
		var err0, err1, err2, err3 error
		event.ConnectionID, err0 = getAttributeString(ev, conntypes.AttributeKeyConnectionID)
		event.ClientID, err1 = getAttributeString(ev, conntypes.AttributeKeyClientID)
		event.CounterpartyClientID, err2 = getAttributeString(ev, conntypes.AttributeKeyCounterpartyClientID)
		event.CounterpartyConnectionID, err3 = getAttributeString(ev, conntypes.AttributeKeyCounterpartyConnectionID)
		if err := errors.Join(err0, err1, err2, err3); err != nil {
			return nil, err
		}
		return &event, nil
	case conntypes.EventTypeConnectionOpenTry:
		var event core.EventConnectionOpenTry
		var err0, err1, err2, err3 error
		event.ConnectionID, err0 = getAttributeString(ev, conntypes.AttributeKeyConnectionID)
		event.ClientID, err1 = getAttributeString(ev, conntypes.AttributeKeyClientID)
		event.CounterpartyClientID, err2 = getAttributeString(ev, conntypes.AttributeKeyCounterpartyClientID)
		event.CounterpartyConnectionID, err3 = getAttributeString(ev, conntypes.AttributeKeyCounterpartyConnectionID)
		if err := errors.Join(err0, err1, err2, err3); err != nil {
			return nil, err
		}
		return &event, nil
	case conntypes.EventTypeConnectionOpenAck:
		var event core.EventConnectionOpenAck
		var err0, err1, err2, err3 error
		event.ConnectionID, err0 = getAttributeString(ev, conntypes.AttributeKeyConnectionID)
		event.ClientID, err1 = getAttributeString(ev, conntypes.AttributeKeyClientID)
		event.CounterpartyClientID, err2 = getAttributeString(ev, conntypes.AttributeKeyCounterpartyClientID)
		event.CounterpartyConnectionID, err3 = getAttributeString(ev, conntypes.AttributeKeyCounterpartyConnectionID)
		if err := errors.Join(err0, err1, err2, err3); err != nil {
			return nil, err
		}
		return &event, nil
	case conntypes.EventTypeConnectionOpenConfirm:
		var event core.EventConnectionOpenConfirm
		var err0, err1, err2, err3 error
		event.ConnectionID, err0 = getAttributeString(ev, conntypes.AttributeKeyConnectionID)
		event.ClientID, err1 = getAttributeString(ev, conntypes.AttributeKeyClientID)
		event.CounterpartyClientID, err2 = getAttributeString(ev, conntypes.AttributeKeyCounterpartyClientID)
		event.CounterpartyConnectionID, err3 = getAttributeString(ev, conntypes.AttributeKeyCounterpartyConnectionID)
		if err := errors.Join(err0, err1, err2, err3); err != nil {
			return nil, err
		}
		return &event, nil
	case chantypes.EventTypeChannelOpenInit:
		var event core.EventChannelOpenInit
		var err0, err1, err2, err3, err4 error
		event.PortID, err0 = getAttributeString(ev, chantypes.AttributeKeyPortID)
		event.ChannelID, err1 = getAttributeString(ev, chantypes.AttributeKeyChannelID)
		event.CounterpartyPortID, err2 = getAttributeString(ev, chantypes.AttributeCounterpartyPortID)
		event.ConnectionID, err3 = getAttributeString(ev, chantypes.AttributeKeyConnectionID)
		event.Version, err4 = getAttributeString(ev, chantypes.AttributeVersion)
		if err := errors.Join(err0, err1, err2, err3, err4); err != nil {
			return nil, err
		}
		return &event, nil
	case chantypes.EventTypeChannelOpenTry:
		var event core.EventChannelOpenTry
		var err0, err1, err2, err3, err4, err5 error
		event.PortID, err0 = getAttributeString(ev, chantypes.AttributeKeyPortID)
		event.ChannelID, err1 = getAttributeString(ev, chantypes.AttributeKeyChannelID)
		event.CounterpartyPortID, err2 = getAttributeString(ev, chantypes.AttributeCounterpartyPortID)
		event.CounterpartyChannelID, err3 = getAttributeString(ev, chantypes.AttributeCounterpartyChannelID)
		event.ConnectionID, err4 = getAttributeString(ev, chantypes.AttributeKeyConnectionID)
		event.Version, err5 = getAttributeString(ev, chantypes.AttributeVersion)
		if err := errors.Join(err0, err1, err2, err3, err4, err5); err != nil {
			return nil, err
		}
		return &event, nil
	case chantypes.EventTypeChannelOpenAck:
		var event core.EventChannelOpenAck
		var err0, err1, err2, err3, err4 error
		event.PortID, err0 = getAttributeString(ev, chantypes.AttributeKeyPortID)
		event.ChannelID, err1 = getAttributeString(ev, chantypes.AttributeKeyChannelID)
		event.CounterpartyPortID, err2 = getAttributeString(ev, chantypes.AttributeCounterpartyPortID)
		event.CounterpartyChannelID, err3 = getAttributeString(ev, chantypes.AttributeCounterpartyChannelID)
		event.ConnectionID, err4 = getAttributeString(ev, chantypes.AttributeKeyConnectionID)
		if err := errors.Join(err0, err1, err2, err3, err4); err != nil {
			return nil, err
		}
		return &event, nil
	case chantypes.EventTypeChannelOpenConfirm:
		var event core.EventChannelOpenConfirm
		var err0, err1, err2, err3, err4 error
		event.PortID, err0 = getAttributeString(ev, chantypes.AttributeKeyPortID)
		event.ChannelID, err1 = getAttributeString(ev, chantypes.AttributeKeyChannelID)
		event.CounterpartyPortID, err2 = getAttributeString(ev, chantypes.AttributeCounterpartyPortID)
		event.CounterpartyChannelID, err3 = getAttributeString(ev, chantypes.AttributeCounterpartyChannelID)
		event.ConnectionID, err4 = getAttributeString(ev, chantypes.AttributeKeyConnectionID)
		if err := errors.Join(err0, err1, err2, err3, err4); err != nil {
			return nil, err
		}
		return &event, nil
	case chantypes.EventTypeChannelCloseInit:
		var event core.EventChannelCloseInit
		var err0, err1, err2, err3, err4 error
		event.PortID, err0 = getAttributeString(ev, chantypes.AttributeKeyPortID)
		event.ChannelID, err1 = getAttributeString(ev, chantypes.AttributeKeyChannelID)
		event.CounterpartyPortID, err2 = getAttributeString(ev, chantypes.AttributeCounterpartyPortID)
		event.CounterpartyChannelID, err3 = getAttributeString(ev, chantypes.AttributeCounterpartyChannelID)
		event.ConnectionID, err4 = getAttributeString(ev, chantypes.AttributeKeyConnectionID)
		if err := errors.Join(err0, err1, err2, err3, err4); err != nil {
			return nil, err
		}
		return &event, nil
	case chantypes.EventTypeChannelCloseConfirm:
		var event core.EventChannelCloseConfirm
		var err0, err1, err2, err3, err4 error
		event.PortID, err0 = getAttributeString(ev, chantypes.AttributeKeyPortID)
		event.ChannelID, err1 = getAttributeString(ev, chantypes.AttributeKeyChannelID)
		event.CounterpartyPortID, err2 = getAttributeString(ev, chantypes.AttributeCounterpartyPortID)
		event.CounterpartyChannelID, err3 = getAttributeString(ev, chantypes.AttributeCounterpartyChannelID)
		event.ConnectionID, err4 = getAttributeString(ev, chantypes.AttributeKeyConnectionID)
		if err := errors.Join(err0, err1, err2, err3, err4); err != nil {
			return nil, err
		}
		return &event, nil
	case chantypes.EventTypeSendPacket:
		var event core.EventSendPacket
		var err0, err1, err2, err3, err4, err5, err6, err7, err8, err9 error
		event.Data, err0 = getAttributeBytes(ev, chantypes.AttributeKeyDataHex)
		event.TimeoutHeight, err1 = getAttributeHeight(ev, chantypes.AttributeKeyTimeoutHeight)
		event.TimeoutTimestamp, err2 = getAttributeTimestamp(ev, chantypes.AttributeKeyTimeoutTimestamp)
		event.Sequence, err3 = getAttributeUint64(ev, chantypes.AttributeKeySequence)
		event.SrcPort, err4 = getAttributeString(ev, chantypes.AttributeKeySrcPort)
		event.SrcChannel, err5 = getAttributeString(ev, chantypes.AttributeKeySrcChannel)
		event.DstPort, err6 = getAttributeString(ev, chantypes.AttributeKeyDstPort)
		event.DstChannel, err7 = getAttributeString(ev, chantypes.AttributeKeyDstChannel)
		event.ChannelOrdering, err8 = getAttributeOrder(ev, chantypes.AttributeKeyChannelOrdering)
		event.ConnectionID, err9 = getAttributeString(ev, chantypes.AttributeKeyConnectionID)
		if err := errors.Join(err0, err1, err2, err3, err4, err5, err6, err7, err8, err9); err != nil {
			return nil, err
		}
		return &event, nil
	case chantypes.EventTypeRecvPacket:
		var event core.EventRecvPacket
		var err0, err1, err2, err3, err4, err5, err6, err7, err8, err9 error
		event.Data, err0 = getAttributeBytes(ev, chantypes.AttributeKeyDataHex)
		event.TimeoutHeight, err1 = getAttributeHeight(ev, chantypes.AttributeKeyTimeoutHeight)
		event.TimeoutTimestamp, err2 = getAttributeTimestamp(ev, chantypes.AttributeKeyTimeoutTimestamp)
		event.Sequence, err3 = getAttributeUint64(ev, chantypes.AttributeKeySequence)
		event.SrcPort, err4 = getAttributeString(ev, chantypes.AttributeKeySrcPort)
		event.SrcChannel, err5 = getAttributeString(ev, chantypes.AttributeKeySrcChannel)
		event.DstPort, err6 = getAttributeString(ev, chantypes.AttributeKeyDstPort)
		event.DstChannel, err7 = getAttributeString(ev, chantypes.AttributeKeyDstChannel)
		event.ChannelOrdering, err8 = getAttributeOrder(ev, chantypes.AttributeKeyChannelOrdering)
		event.ConnectionID, err9 = getAttributeString(ev, chantypes.AttributeKeyConnectionID)
		if err := errors.Join(err0, err1, err2, err3, err4, err5, err6, err7, err8, err9); err != nil {
			return nil, err
		}
		return &event, nil
	case chantypes.EventTypeWriteAck:
		var event core.EventWriteAck
		var err0, err1, err2, err3, err4, err5, err6, err7, err8, err9 error
		event.Data, err0 = getAttributeBytes(ev, chantypes.AttributeKeyDataHex)
		event.TimeoutHeight, err1 = getAttributeHeight(ev, chantypes.AttributeKeyTimeoutHeight)
		event.TimeoutTimestamp, err2 = getAttributeTimestamp(ev, chantypes.AttributeKeyTimeoutTimestamp)
		event.Sequence, err3 = getAttributeUint64(ev, chantypes.AttributeKeySequence)
		event.SrcPort, err4 = getAttributeString(ev, chantypes.AttributeKeySrcPort)
		event.SrcChannel, err5 = getAttributeString(ev, chantypes.AttributeKeySrcChannel)
		event.DstPort, err6 = getAttributeString(ev, chantypes.AttributeKeyDstPort)
		event.DstChannel, err7 = getAttributeString(ev, chantypes.AttributeKeyDstChannel)
		event.Ack, err8 = getAttributeBytes(ev, chantypes.AttributeKeyAckHex)
		event.ConnectionID, err9 = getAttributeString(ev, chantypes.AttributeKeyConnectionID)
		if err := errors.Join(err0, err1, err2, err3, err4, err5, err6, err7, err8, err9); err != nil {
			return nil, err
		}
		return &event, nil
	case chantypes.EventTypeAcknowledgePacket:
		var event core.EventAcknowledgePacket
		var err0, err1, err2, err3, err4, err5, err6, err7, err8 error
		event.TimeoutHeight, err0 = getAttributeHeight(ev, chantypes.AttributeKeyTimeoutHeight)
		event.TimeoutTimestamp, err1 = getAttributeTimestamp(ev, chantypes.AttributeKeyTimeoutTimestamp)
		event.Sequence, err2 = getAttributeUint64(ev, chantypes.AttributeKeySequence)
		event.SrcPort, err3 = getAttributeString(ev, chantypes.AttributeKeySrcPort)
		event.SrcChannel, err4 = getAttributeString(ev, chantypes.AttributeKeySrcChannel)
		event.DstPort, err5 = getAttributeString(ev, chantypes.AttributeKeyDstPort)
		event.DstChannel, err6 = getAttributeString(ev, chantypes.AttributeKeyDstChannel)
		event.ChannelOrdering, err7 = getAttributeOrder(ev, chantypes.AttributeKeyChannelOrdering)
		event.ConnectionID, err8 = getAttributeString(ev, chantypes.AttributeKeyConnectionID)
		if err := errors.Join(err0, err1, err2, err3, err4, err5, err6, err7, err8); err != nil {
			return nil, err
		}
		return &event, nil
	case chantypes.EventTypeTimeoutPacket:
		var event core.EventTimeoutPacket
		var err0, err1, err2, err3, err4, err5, err6, err7, err8 error
		event.TimeoutHeight, err0 = getAttributeHeight(ev, chantypes.AttributeKeyTimeoutHeight)
		event.TimeoutTimestamp, err1 = getAttributeTimestamp(ev, chantypes.AttributeKeyTimeoutTimestamp)
		event.Sequence, err2 = getAttributeUint64(ev, chantypes.AttributeKeySequence)
		event.SrcPort, err3 = getAttributeString(ev, chantypes.AttributeKeySrcPort)
		event.SrcChannel, err4 = getAttributeString(ev, chantypes.AttributeKeySrcChannel)
		event.DstPort, err5 = getAttributeString(ev, chantypes.AttributeKeyDstPort)
		event.DstChannel, err6 = getAttributeString(ev, chantypes.AttributeKeyDstChannel)
		event.ConnectionID, err7 = getAttributeString(ev, chantypes.AttributeKeyConnectionID)
		event.ChannelOrdering, err8 = getAttributeOrder(ev, chantypes.AttributeKeyChannelOrdering)
		if err := errors.Join(err0, err1, err2, err3, err4, err5, err6, err7, err8); err != nil {
			return nil, err
		}
		return &event, nil
	case chantypes.EventTypeChannelClosed:
		var event core.EventChannelClosed
		var err0, err1, err2, err3, err4, err5 error
		event.PortID, err0 = getAttributeString(ev, chantypes.AttributeKeyPortID)
		event.ChannelID, err1 = getAttributeString(ev, chantypes.AttributeKeyChannelID)
		event.CounterpartyPortID, err2 = getAttributeString(ev, chantypes.AttributeCounterpartyPortID)
		event.CounterpartyChannelID, err3 = getAttributeString(ev, chantypes.AttributeCounterpartyChannelID)
		event.ConnectionID, err4 = getAttributeString(ev, chantypes.AttributeKeyConnectionID)
		event.ChannelOrdering, err5 = getAttributeOrder(ev, chantypes.AttributeKeyChannelOrdering)
		if err := errors.Join(err0, err1, err2, err3, err4, err5); err != nil {
			return nil, err
		}
		return &event, nil
	default:
		return &UnparseableEventLog{ev}, nil
	}
}

func getAttributeString(ev sdk.StringEvent, key string) (string, error) {
	for _, attr := range ev.Attributes {
		if attr.Key == key {
			return attr.Value, nil
		}
	}
	return "", fmt.Errorf("failed to find attribute of key %q", key)
}

func getAttributeBytes(ev sdk.StringEvent, key string) ([]byte, error) {
	v, err := getAttributeString(ev, key)
	if err != nil {
		return nil, err
	}
	bz, err := hex.DecodeString(v)
	if err != nil {
		return nil, fmt.Errorf("failed to decode hex string: %v", err)
	}
	return bz, nil
}

func getAttributeHeight(ev sdk.StringEvent, key string) (clienttypes.Height, error) {
	v, err := getAttributeString(ev, key)
	if err != nil {
		return clienttypes.Height{}, err
	}
	height, err := clienttypes.ParseHeight(v)
	if err != nil {
		return clienttypes.Height{}, fmt.Errorf("failed to parse height: %v", err)
	}
	return height, nil
}

func getAttributeUint64(ev sdk.StringEvent, key string) (uint64, error) {
	v, err := getAttributeString(ev, key)
	if err != nil {
		return 0, err
	}
	d, err := strconv.ParseUint(v, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse uint: %v", err)
	}
	return d, nil
}

func getAttributeHeights(ev sdk.StringEvent, key string) ([]clienttypes.Height, error) {
	v, err := getAttributeString(ev, key)
	if err != nil {
		return nil, err
	}
	var heights []clienttypes.Height
	for _, s := range strings.Split(v, ",") {
		height, err := clienttypes.ParseHeight(s)
		if err != nil {
			return nil, fmt.Errorf("failed to parse height: %v", err)
		}
		heights = append(heights, height)
	}
	return heights, nil
}

func getAttributeClientMessage(cdc codec.Codec, ev sdk.StringEvent, key string) (exported.ClientMessage, error) {
	bz, err := getAttributeBytes(ev, key)
	if err != nil {
		return nil, err
	}
	cliMsg, err := clienttypes.UnmarshalClientMessage(cdc, bz)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal client message: %v", err)
	}
	return cliMsg, nil
}

func getAttributeTimestamp(ev sdk.StringEvent, key string) (time.Time, error) {
	d, err := getAttributeUint64(ev, key)
	if err != nil {
		return time.Time{}, err
	}
	return time.Unix(0, int64(d)), nil
}

func getAttributeOrder(ev sdk.StringEvent, key string) (chantypes.Order, error) {
	v, err := getAttributeString(ev, key)
	if err != nil {
		return 0, err
	}
	order, found := chantypes.Order_value[v]
	if !found {
		return 0, fmt.Errorf("invalid order enum: %v", v)
	}
	return chantypes.Order(order), nil
}
