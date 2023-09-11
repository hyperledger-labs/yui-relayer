package core

import (
	"time"

	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
)

type MsgID interface {
	is_MsgID()
}

type IsMsgID struct{}

func (IsMsgID) is_MsgID() {}

var _ MsgID = IsMsgID{}

type MsgResult interface {
	BlockHeight() clienttypes.Height
	Status() (bool, string)
	Events() []MsgEventLog
}

type MsgEventLog interface {
	is_MsgEventLog()
}

type isMsgEventLog struct{}

func (isMsgEventLog) is_MsgEventLog() {}

var (
	_ MsgEventLog = isMsgEventLog{}
	_ MsgEventLog = (*EventGenerateClientIdentifier)(nil)
	_ MsgEventLog = (*EventGenerateConnectionIdentifier)(nil)
	_ MsgEventLog = (*EventGenerateChannelIdentifier)(nil)
	_ MsgEventLog = (*EventSendPacket)(nil)
	_ MsgEventLog = (*EventRecvPacket)(nil)
	_ MsgEventLog = (*EventWriteAcknowledgement)(nil)
	_ MsgEventLog = (*EventAcknowledgePacket)(nil)
	_ MsgEventLog = (*EventUnknown)(nil)
)

type EventGenerateClientIdentifier struct {
	isMsgEventLog

	ID string
}

type EventGenerateConnectionIdentifier struct {
	isMsgEventLog

	ID string
}

type EventGenerateChannelIdentifier struct {
	isMsgEventLog

	ID string
}

type EventSendPacket struct {
	isMsgEventLog

	Sequence         uint64
	SrcPort          string
	SrcChannel       string
	TimeoutHeight    clienttypes.Height
	TimeoutTimestamp time.Time
	Data             []byte
}

type EventRecvPacket struct {
	isMsgEventLog

	Sequence         uint64
	DstPort          string
	DstChannel       string
	TimeoutHeight    clienttypes.Height
	TimeoutTimestamp time.Time
	Data             []byte
}

type EventWriteAcknowledgement struct {
	isMsgEventLog

	Sequence        uint64
	DstPort         string
	DstChannel      string
	Acknowledgement []byte
}

type EventAcknowledgePacket struct {
	isMsgEventLog

	Sequence         uint64
	SrcPort          string
	SrcChannel       string
	TimeoutHeight    clienttypes.Height
	TimeoutTimestamp time.Time
}

type EventUnknown struct {
	isMsgEventLog

	Value any
}
