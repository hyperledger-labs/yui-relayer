package core

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"
)

// RelayMsgs contains the msgs that need to be sent to both a src and dst chain
// after a given relay round. MaxTxSize and MaxMsgLength are ignored if they are
// set to zero.
type RelayMsgs struct {
	Src          []sdk.Msg `json:"src"`
	Dst          []sdk.Msg `json:"dst"`
	MaxTxSize    uint64    `json:"max_tx_size"`    // maximum permitted size of the msgs in a bundled relay transaction
	MaxMsgLength uint64    `json:"max_msg_length"` // maximum amount of messages in a bundled relay transaction

	Last      bool `json:"last"`
	Succeeded bool `json:"success"`

	SrcMsgIDs []MsgID `json:"src_msg_ids"`
	DstMsgIDs []MsgID `json:"dst_msg_ids"`
}

// NewRelayMsgs returns an initialized version of relay messages
func NewRelayMsgs() *RelayMsgs {
	return &RelayMsgs{Src: []sdk.Msg{}, Dst: []sdk.Msg{}, Last: false, Succeeded: false}
}

// Ready returns true if there are messages to relay
func (r *RelayMsgs) Ready() bool {
	return r != nil && (len(r.Src) > 0 || len(r.Dst) > 0)
}

// Success returns the success var
func (r *RelayMsgs) Success() bool {
	return r.Succeeded
}

func (r *RelayMsgs) IsMaxTx(msgLen, txSize uint64) bool {
	return (r.MaxMsgLength != 0 && msgLen > r.MaxMsgLength) ||
		(r.MaxTxSize != 0 && txSize > r.MaxTxSize)
}

// Send sends the messages with appropriate output
// TODO: Parallelize? Maybe?
func (r *RelayMsgs) Send(src, dst Chain) {
	logger := GetChannelPairLogger(src, dst)
	//nolint:prealloc // can not be pre allocated
	var (
		msgLen, txSize uint64
		msgs           []sdk.Msg
	)

	r.Succeeded = true

	srcMsgIDs := make([]MsgID, len(r.Src))
	dstMsgIDs := make([]MsgID, len(r.Dst))
	// submit batches of relay transactions
	maxTxCount := 0

	for _, msg := range r.Src {
		bz, err := proto.Marshal(msg)
		if err != nil {
			logger.Error("failed to marshal msg", err)
			panic(err)
		}

		msgLen++
		txSize += uint64(len(bz))

		if r.IsMaxTx(msgLen, txSize) {
			// Submit the transactions to src chain and update its status
			msgIDs, err := src.SendMsgs(msgs)
			if err != nil {
				logger.Error("failed to send msgs", err, "msgs", msgs)
			}
			r.Succeeded = r.Succeeded && (err == nil)
			if err == nil {
				for i := range msgs {
					srcMsgIDs[i+maxTxCount] = msgIDs[i]
				}
			}
			// clear the current batch and reset variables
			maxTxCount += len(msgs)
			msgLen, txSize = 1, uint64(len(bz))
			msgs = []sdk.Msg{}
		}
		msgs = append(msgs, msg)
	}

	// submit leftover msgs
	if len(msgs) > 0 {
		msgIDs, err := src.SendMsgs(msgs)
		if err != nil {
			logger.Error("failed to send msgs", err, "msgs", msgs)
		}
		r.Succeeded = r.Succeeded && (err == nil)
		if err == nil {
			for i := range msgs {
				srcMsgIDs[i+maxTxCount] = msgIDs[i]
			}
		}
	}

	// reset variables
	msgLen, txSize = 0, 0
	msgs = []sdk.Msg{}
	maxTxCount = 0

	for _, msg := range r.Dst {
		bz, err := proto.Marshal(msg)
		if err != nil {
			logger.Error("failed to marshal msg", err)
			panic(err)
		}

		msgLen++
		txSize += uint64(len(bz))

		if r.IsMaxTx(msgLen, txSize) {
			// Submit the transaction to dst chain and update its status
			msgIDs, err := dst.SendMsgs(msgs)
			if err != nil {
				logger.Error("failed to send msgs", err, "msgs", msgs)
			}
			r.Succeeded = r.Succeeded && (err == nil)
			if err == nil {
				for i := range msgs {
					dstMsgIDs[i+maxTxCount] = msgIDs[i]
				}
			}
			// clear the current batch and reset variables
			maxTxCount += len(msgs)
			msgLen, txSize = 1, uint64(len(bz))
			msgs = []sdk.Msg{}
		}
		msgs = append(msgs, msg)
	}

	// submit leftover msgs
	if len(msgs) > 0 {
		msgIDs, err := dst.SendMsgs(msgs)
		if err != nil {
			logger.Error("failed to send msgs", err, "msgs", msgs)
		}
		r.Succeeded = r.Succeeded && (err == nil)
		if err == nil {
			for i := range msgs {
				dstMsgIDs[i+maxTxCount] = msgIDs[i]
			}
		}
	}
	r.SrcMsgIDs = srcMsgIDs
	r.DstMsgIDs = dstMsgIDs
}

// Merge merges the argument into the receiver
func (r *RelayMsgs) Merge(other *RelayMsgs) {
	r.Src = append(r.Src, other.Src...)
	r.Dst = append(r.Dst, other.Dst...)
}
