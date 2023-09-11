package tendermint

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	conntypes "github.com/cosmos/ibc-go/v7/modules/core/03-connection/types"
	chantypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	"github.com/hyperledger-labs/yui-relayer/core"
)

var (
	_ core.MsgID     = (*MsgID)(nil)
	_ core.MsgResult = (*MsgResult)(nil)
)

type MsgID struct {
	core.IsMsgID

	txHash   string
	msgIndex uint32
}

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

func parseMsgEventLogs(logs sdk.ABCIMessageLogs, msgIndex uint32) ([]core.MsgEventLog, error) {
	var msgEventLogs []core.MsgEventLog
	for _, log := range logs {
		if msgIndex == log.MsgIndex {
			for _, ev := range log.Events {
				event, err := parseMsgEventLog(ev)
				if err != nil {
					return nil, fmt.Errorf("failed to parse msg event log: %v", err)
				}
				msgEventLogs = append(msgEventLogs, event)
			}
		}
	}
	return msgEventLogs, nil
}

func parseMsgEventLog(ev sdk.StringEvent) (core.MsgEventLog, error) {
	switch ev.Type {
	case clienttypes.EventTypeCreateClient:
		clientID, err := getAttributeString(ev, clienttypes.AttributeKeyClientID)
		if err != nil {
			return nil, err
		}
		return &core.EventGenerateClientIdentifier{ID: clientID}, nil
	case conntypes.EventTypeConnectionOpenInit, conntypes.EventTypeConnectionOpenTry:
		connectionID, err := getAttributeString(ev, conntypes.AttributeKeyConnectionID)
		if err != nil {
			return nil, err
		}
		return &core.EventGenerateConnectionIdentifier{ID: connectionID}, nil
	case chantypes.EventTypeChannelOpenInit, chantypes.EventTypeChannelOpenTry:
		channelID, err := getAttributeString(ev, chantypes.AttributeKeyChannelID)
		if err != nil {
			return nil, err
		}
		return &core.EventGenerateChannelIdentifier{ID: channelID}, nil
	case chantypes.EventTypeSendPacket:
		var event core.EventSendPacket
		var err0, err1, err2, err3, err4, err5 error
		event.Sequence, err0 = getAttributeUint64(ev, chantypes.AttributeKeySequence)
		event.SrcPort, err1 = getAttributeString(ev, chantypes.AttributeKeySrcPort)
		event.SrcChannel, err2 = getAttributeString(ev, chantypes.AttributeKeySrcChannel)
		event.TimeoutHeight, err3 = getAttributeHeight(ev, chantypes.AttributeKeyTimeoutHeight)
		event.TimeoutTimestamp, err4 = getAttributeTimestamp(ev, chantypes.AttributeKeyTimeoutTimestamp)
		event.Data, err5 = getAttributeBytes(ev, chantypes.AttributeKeyDataHex)
		if err := errors.Join(err0, err1, err2, err3, err4, err5); err != nil {
			return nil, err
		}
		return &event, nil
	case chantypes.EventTypeRecvPacket:
		var event core.EventRecvPacket
		var err0, err1, err2, err3, err4, err5 error
		event.Sequence, err0 = getAttributeUint64(ev, chantypes.AttributeKeySequence)
		event.DstPort, err1 = getAttributeString(ev, chantypes.AttributeKeyDstPort)
		event.DstChannel, err2 = getAttributeString(ev, chantypes.AttributeKeyDstChannel)
		event.TimeoutHeight, err3 = getAttributeHeight(ev, chantypes.AttributeKeyTimeoutHeight)
		event.TimeoutTimestamp, err4 = getAttributeTimestamp(ev, chantypes.AttributeKeyTimeoutTimestamp)
		event.Data, err5 = getAttributeBytes(ev, chantypes.AttributeKeyDataHex)
		if err := errors.Join(err0, err1, err2, err3, err4, err5); err != nil {
			return nil, err
		}
		return &event, nil
	case chantypes.EventTypeWriteAck:
		var event core.EventWriteAcknowledgement
		var err0, err1, err2, err3 error
		event.Sequence, err0 = getAttributeUint64(ev, chantypes.AttributeKeySequence)
		event.DstPort, err1 = getAttributeString(ev, chantypes.AttributeKeyDstPort)
		event.DstChannel, err2 = getAttributeString(ev, chantypes.AttributeKeyDstChannel)
		event.Acknowledgement, err3 = getAttributeBytes(ev, chantypes.AttributeKeyAckHex)
		if err := errors.Join(err0, err1, err2, err3); err != nil {
			return nil, err
		}
		return &event, nil
	case chantypes.EventTypeAcknowledgePacket:
		var event core.EventAcknowledgePacket
		var err0, err1, err2, err3, err4 error
		event.Sequence, err0 = getAttributeUint64(ev, chantypes.AttributeKeySequence)
		event.SrcPort, err1 = getAttributeString(ev, chantypes.AttributeKeySrcPort)
		event.SrcChannel, err2 = getAttributeString(ev, chantypes.AttributeKeySrcChannel)
		event.TimeoutHeight, err3 = getAttributeHeight(ev, chantypes.AttributeKeyTimeoutHeight)
		event.TimeoutTimestamp, err4 = getAttributeTimestamp(ev, chantypes.AttributeKeyTimeoutTimestamp)
		if err := errors.Join(err0, err1, err2, err3, err4); err != nil {
			return nil, err
		}
		return &event, nil
	default:
		return &core.EventUnknown{Value: ev}, nil
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

func getAttributeTimestamp(ev sdk.StringEvent, key string) (time.Time, error) {
	d, err := getAttributeUint64(ev, key)
	if err != nil {
		return time.Time{}, err
	}
	return time.Unix(0, int64(d)), nil
}
