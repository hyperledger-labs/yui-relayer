package core

import (
	"fmt"

	retry "github.com/avast/retry-go"
	"github.com/cosmos/cosmos-sdk/types"
)

// SendMsgsAndCheckResult is an utility function that executes `Chain::SendMsgs` and checks the results of all the messages.
func SendMsgsAndCheckResult(chain Chain, msgs []types.Msg) error {
	ids, err := chain.SendMsgs(msgs)
	if err != nil {
		return fmt.Errorf("failed to send messages: %v", err)
	}
	for i, id := range ids {
		res, err := chain.GetMsgResult(id)
		if err != nil {
			return fmt.Errorf("failed to get the result of msg(%v): %v", msgs[i], err)
		} else if ok, reason := res.Status(); !ok {
			return fmt.Errorf("msg(%v) was successfully broadcasted, but its execution failed: failure_reason=%v", msgs[i], reason)
		}
	}
	return nil
}

// GetFinalizedMsgResult is an utility function that waits for the finalization of the message execution and then returns the result.
func GetFinalizedMsgResult(chain ProvableChain, msgID MsgID) (MsgResult, error) {
	var msgRes MsgResult
	if err := retry.Do(func() error {
		var err error

		// query LFH for each retry because it can proceed.
		header, err := chain.GetLatestFinalizedHeader()
		if err != nil {
			return fmt.Errorf("failed to get latest finalized header: %v", err)
		}

		// query MsgResult for each retry because it can be included in a different block because of reorg
		msgRes, err = chain.GetMsgResult(msgID)
		if err != nil {
			return fmt.Errorf("failed to get messge result: %v", err)
		}

		// check whether the block that includes the message has been finalized, or not
		if msgRes.BlockHeight().GT(header.GetHeight()) {
			return fmt.Errorf("message_height(%v) > latest_finalized_height(%v)", msgRes.BlockHeight(), header.GetHeight())
		}

		return nil
	}, rtyAtt, rtyDel, rtyErr); err != nil {
		return nil, err
	} else {
		return msgRes, nil
	}
}
