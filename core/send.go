package core

import (
	"fmt"
	"time"

	retry "github.com/avast/retry-go"
	"github.com/cosmos/cosmos-sdk/types"
)

// SendCheckMsgs is an utility function that executes `Chain::SendMsgs` and checks the execution results of all the messages.
func SendCheckMsgs(chain Chain, msgs []types.Msg) bool {
	if _, err := chain.SendMsgs(msgs); err != nil {
		GetChainLogger(chain).Error("failed to send msgs", err)
		return false
	}
	return true
}

// GetFinalizedMsgResult is an utility function that waits for the finalization of the message execution and then returns the result.
func GetFinalizedMsgResult(chain ProvableChain, msgID MsgID, retryInterval time.Duration, maxRetry uint) (MsgResult, error) {
	var msgRes MsgResult

	if err := retry.Do(func() error {
		// query LFH for each retry because it can proceed.
		lfHeader, err := chain.GetLatestFinalizedHeader()
		if err != nil {
			return fmt.Errorf("failed to get latest finalized header: %v", err)
		}

		// query MsgResult for each retry because it can be included in a different block because of reorg
		msgRes, err = chain.GetMsgResult(msgID)
		if err != nil {
			return retry.Unrecoverable(fmt.Errorf("failed to get message result: %v", err))
		} else if ok, failureReason := msgRes.Status(); !ok {
			return retry.Unrecoverable(fmt.Errorf("msg(id=%v) execution failed: %v", msgID, failureReason))
		}

		// check whether the block that includes the message has been finalized, or not
		if msgHeight, lfHeight := msgRes.BlockHeight(), lfHeader.GetHeight(); msgHeight.GT(lfHeight) {
			return fmt.Errorf("msg(id=%v) not finalied: msg.height(%v) > lfh.height(%v)", msgID, msgHeight, lfHeight)
		}

		return nil
	}, retry.Attempts(maxRetry), retry.Delay(retryInterval), rtyErr); err != nil {
		return nil, err
	}

	return msgRes, nil
}
