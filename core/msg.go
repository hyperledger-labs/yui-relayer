package core

import (
	"time"

	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
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
	isMsgEventLog()
}

var (
	_ MsgEventLog = (*EventGenerateClientIdentifier)(nil)
	_ MsgEventLog = (*EventGenerateConnectionIdentifier)(nil)
	_ MsgEventLog = (*EventGenerateChannelIdentifier)(nil)
	_ MsgEventLog = (*EventSendPacket)(nil)
	_ MsgEventLog = (*EventRecvPacket)(nil)
	_ MsgEventLog = (*EventWriteAcknowledgement)(nil)
	_ MsgEventLog = (*EventAcknowledgePacket)(nil)
	_ MsgEventLog = (*EventUnknown)(nil)
)

func (*EventGenerateClientIdentifier) isMsgEventLog()     {}
func (*EventGenerateConnectionIdentifier) isMsgEventLog() {}
func (*EventGenerateChannelIdentifier) isMsgEventLog()    {}
func (*EventSendPacket) isMsgEventLog()                   {}
func (*EventRecvPacket) isMsgEventLog()                   {}
func (*EventWriteAcknowledgement) isMsgEventLog()         {}
func (*EventAcknowledgePacket) isMsgEventLog()            {}
func (*EventUnknown) isMsgEventLog()                      {}

type EventGenerateClientIdentifier struct {
	ID string
}

type EventGenerateConnectionIdentifier struct {
	ID string
}

type EventGenerateChannelIdentifier struct {
	ID string
}

type EventSendPacket struct {
	Sequence         uint64
	SrcPort          string
	SrcChannel       string
	TimeoutHeight    clienttypes.Height
	TimeoutTimestamp time.Time
	Data             []byte
}

type EventRecvPacket struct {
	Sequence         uint64
	DstPort          string
	DstChannel       string
	TimeoutHeight    clienttypes.Height
	TimeoutTimestamp time.Time
	Data             []byte
}

type EventWriteAcknowledgement struct {
	Sequence        uint64
	DstPort         string
	DstChannel      string
	Acknowledgement []byte
}

type EventAcknowledgePacket struct {
	Sequence         uint64
	SrcPort          string
	SrcChannel       string
	TimeoutHeight    clienttypes.Height
	TimeoutTimestamp time.Time
}

type EventUnknown struct {
	Value any
}
