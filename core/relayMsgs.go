package core

import (
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/gogo/protobuf/proto"
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
}

// NewRelayMsgs returns an initialized version of relay messages
func NewRelayMsgs() *RelayMsgs {
	return &RelayMsgs{Src: []sdk.Msg{}, Dst: []sdk.Msg{}, Last: false, Succeeded: false}
}

// Ready returns true if there are messages to relay
func (r *RelayMsgs) Ready() bool {
	if r == nil {
		return false
	}

	if len(r.Src) == 0 && len(r.Dst) == 0 {
		return false
	}
	return true
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
func (r *RelayMsgs) Send(src, dst ChainI) {
	//nolint:prealloc // can not be pre allocated
	var (
		msgLen, txSize uint64
		msgs           []sdk.Msg
	)

	r.Succeeded = true

	// submit batches of relay transactions
	for _, msg := range r.Src {
		bz, err := proto.Marshal(msg)
		if err != nil {
			panic(err)
		}

		msgLen++
		txSize += uint64(len(bz))

		if r.IsMaxTx(msgLen, txSize) {
			// Submit the transactions to src chain and update its status
			r.Succeeded = r.Succeeded && send(src, msgs)

			// clear the current batch and reset variables
			msgLen, txSize = 1, uint64(len(bz))
			msgs = []sdk.Msg{}
		}
		msgs = append(msgs, msg)
	}

	// submit leftover msgs
	if len(msgs) > 0 && !send(src, msgs) {
		r.Succeeded = false
	}

	// reset variables
	msgLen, txSize = 0, 0
	msgs = []sdk.Msg{}

	for _, msg := range r.Dst {
		bz, err := proto.Marshal(msg)
		if err != nil {
			panic(err)
		}

		msgLen++
		txSize += uint64(len(bz))

		if r.IsMaxTx(msgLen, txSize) {
			// Submit the transaction to dst chain and update its status
			r.Succeeded = r.Succeeded && send(dst, msgs)

			// clear the current batch and reset variables
			msgLen, txSize = 1, uint64(len(bz))
			msgs = []sdk.Msg{}
		}
		msgs = append(msgs, msg)
	}

	// submit leftover msgs
	if len(msgs) > 0 && !send(dst, msgs) {
		r.Succeeded = false
	}
}

// Submits the messages to the provided chain and logs the result of the transaction.
// Returns true upon success and false otherwise.
func send(chain ChainI, msgs []sdk.Msg) bool {
	return chain.Send(msgs)

	// res, err := chain.SendMsgs(msgs)
	// if err != nil || res.Code != 0 {
	// 	chain.LogFailedTx(res, err, msgs)
	// 	return false
	// }
	// // NOTE: Add more data to this such as identifiers
	// chain.LogSuccessTx(res, msgs)

	// return true
}

func GetMsgAction(msgs []sdk.Msg) string {
	var out string
	for i, msg := range msgs {
		out += fmt.Sprintf("%d:%s,", i, msg.Type())
	}
	return strings.TrimSuffix(out, ",")
}
