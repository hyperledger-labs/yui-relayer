package core

import (
	"time"

	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	chantypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
)

type MsgID interface {
	IsMsgID()
}

type MsgResult interface {
	BlockHeight() clienttypes.Height
	Status() (bool, string)
	Events() []MsgEventLog
}

type MsgEventLog interface {
	IsMsgEventLog()
}

var (
	_ MsgEventLog = (*EventCreateClient)(nil)
	_ MsgEventLog = (*EventUpdateClient)(nil)
	_ MsgEventLog = (*EventConnectionOpenInit)(nil)
	_ MsgEventLog = (*EventConnectionOpenTry)(nil)
	_ MsgEventLog = (*EventConnectionOpenAck)(nil)
	_ MsgEventLog = (*EventConnectionOpenConfirm)(nil)
	_ MsgEventLog = (*EventChannelOpenInit)(nil)
	_ MsgEventLog = (*EventChannelOpenTry)(nil)
	_ MsgEventLog = (*EventChannelOpenAck)(nil)
	_ MsgEventLog = (*EventChannelOpenConfirm)(nil)
	_ MsgEventLog = (*EventChannelCloseInit)(nil)
	_ MsgEventLog = (*EventChannelCloseConfirm)(nil)
	_ MsgEventLog = (*EventSendPacket)(nil)
	_ MsgEventLog = (*EventRecvPacket)(nil)
	_ MsgEventLog = (*EventWriteAck)(nil)
	_ MsgEventLog = (*EventAcknowledgePacket)(nil)
	_ MsgEventLog = (*EventTimeoutPacket)(nil)
	_ MsgEventLog = (*EventChannelClosed)(nil)
)

func (EventCreateClient) IsMsgEventLog()          {}
func (EventUpdateClient) IsMsgEventLog()          {}
func (EventConnectionOpenInit) IsMsgEventLog()    {}
func (EventConnectionOpenTry) IsMsgEventLog()     {}
func (EventConnectionOpenAck) IsMsgEventLog()     {}
func (EventConnectionOpenConfirm) IsMsgEventLog() {}
func (EventChannelOpenInit) IsMsgEventLog()       {}
func (EventChannelOpenTry) IsMsgEventLog()        {}
func (EventChannelOpenAck) IsMsgEventLog()        {}
func (EventChannelOpenConfirm) IsMsgEventLog()    {}
func (EventChannelCloseInit) IsMsgEventLog()      {}
func (EventChannelCloseConfirm) IsMsgEventLog()   {}
func (EventSendPacket) IsMsgEventLog()            {}
func (EventRecvPacket) IsMsgEventLog()            {}
func (EventWriteAck) IsMsgEventLog()              {}
func (EventAcknowledgePacket) IsMsgEventLog()     {}
func (EventTimeoutPacket) IsMsgEventLog()         {}
func (EventChannelClosed) IsMsgEventLog()         {}

type EventCreateClient struct {
	ClientID        string
	ClientType      string
	ConsensusHeight clienttypes.Height
}

type EventUpdateClient struct {
	ClientID         string
	ClientType       string
	ConsensusHeights []clienttypes.Height
	ClientMessage    exported.ClientMessage
}

type EventConnectionOpenInit struct {
	ConnectionID             string
	ClientID                 string
	CounterpartyClientID     string
	CounterpartyConnectionID string
}

type EventConnectionOpenTry struct {
	ConnectionID             string
	ClientID                 string
	CounterpartyClientID     string
	CounterpartyConnectionID string
}

type EventConnectionOpenAck struct {
	ConnectionID             string
	ClientID                 string
	CounterpartyClientID     string
	CounterpartyConnectionID string
}

type EventConnectionOpenConfirm struct {
	ConnectionID             string
	ClientID                 string
	CounterpartyClientID     string
	CounterpartyConnectionID string
}

type EventChannelOpenInit struct {
	PortID             string
	ChannelID          string
	CounterpartyPortID string
	ConnectionID       string
	Version            string
}

type EventChannelOpenTry struct {
	PortID                string
	ChannelID             string
	CounterpartyPortID    string
	CounterpartyChannelID string
	ConnectionID          string
	Version               string
}

type EventChannelOpenAck struct {
	PortID                string
	ChannelID             string
	CounterpartyPortID    string
	CounterpartyChannelID string
	ConnectionID          string
}

type EventChannelOpenConfirm struct {
	PortID                string
	ChannelID             string
	CounterpartyPortID    string
	CounterpartyChannelID string
	ConnectionID          string
}

type EventChannelCloseInit struct {
	PortID                string
	ChannelID             string
	CounterpartyPortID    string
	CounterpartyChannelID string
	ConnectionID          string
}

type EventChannelCloseConfirm struct {
	PortID                string
	ChannelID             string
	CounterpartyPortID    string
	CounterpartyChannelID string
	ConnectionID          string
}

type EventSendPacket struct {
	Data             []byte
	TimeoutHeight    clienttypes.Height
	TimeoutTimestamp time.Time
	Sequence         uint64
	SrcPort          string
	SrcChannel       string
	DstPort          string
	DstChannel       string
	ChannelOrdering  chantypes.Order
	ConnectionID     string
}

type EventRecvPacket struct {
	Data             []byte
	TimeoutHeight    clienttypes.Height
	TimeoutTimestamp time.Time
	Sequence         uint64
	SrcPort          string
	SrcChannel       string
	DstPort          string
	DstChannel       string
	ChannelOrdering  chantypes.Order
	ConnectionID     string
}

type EventWriteAck struct {
	Data             []byte
	TimeoutHeight    clienttypes.Height
	TimeoutTimestamp time.Time
	Sequence         uint64
	SrcPort          string
	SrcChannel       string
	DstPort          string
	DstChannel       string
	Ack              []byte
	ConnectionID     string
}

type EventAcknowledgePacket struct {
	TimeoutHeight    clienttypes.Height
	TimeoutTimestamp time.Time
	Sequence         uint64
	SrcPort          string
	SrcChannel       string
	DstPort          string
	DstChannel       string
	ChannelOrdering  chantypes.Order
	ConnectionID     string
}

type EventTimeoutPacket struct {
	TimeoutHeight    clienttypes.Height
	TimeoutTimestamp time.Time
	Sequence         uint64
	SrcPort          string
	SrcChannel       string
	DstPort          string
	DstChannel       string
	ConnectionID     string
	ChannelOrdering  chantypes.Order
}

type EventChannelClosed struct {
	PortID                string
	ChannelID             string
	CounterpartyPortID    string
	CounterpartyChannelID string
	ConnectionID          string
	ChannelOrdering       chantypes.Order
}
