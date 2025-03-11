package core

import (
	"context"
	"fmt"
	"time"

	retry "github.com/avast/retry-go"
)

// GetFinalizedMsgResult is an utility function that waits for the finalization of the message execution and then returns the result.
func GetFinalizedMsgResult(ctx context.Context, chain ProvableChain, msgID MsgID) (MsgResult, error) {
	var msgRes MsgResult

	avgBlockTime := chain.AverageBlockTime()

	if err := retry.Do(func() error {
		// query LFH for each retry because it can proceed.
		lfHeader, err := chain.GetLatestFinalizedHeader(ctx)
		if err != nil {
			return fmt.Errorf("failed to get latest finalized header: %v", err)
		}

		// query MsgResult for each retry because it can be included in a different block because of reorg
		msgRes, err = chain.GetMsgResult(ctx, msgID)
		if err != nil {
			return retry.Unrecoverable(fmt.Errorf("failed to get message result: %v", err))
		} else if ok, failureReason := msgRes.Status(); !ok {
			return retry.Unrecoverable(fmt.Errorf("msg(id=%v) execution failed: %v", msgID, failureReason))
		}

		// check whether the block that includes the message has been finalized, or not
		if msgHeight, lfHeight := msgRes.BlockHeight(), lfHeader.GetHeight(); msgHeight.GT(lfHeight) {
			// wait for the block including the msg to be finalized
			var waitTime time.Duration
			if msgHeight.GetRevisionNumber() != lfHeight.GetRevisionNumber() {
				waitTime = avgBlockTime //TODO: is there better default value?
			} else {
				waitTime = avgBlockTime * time.Duration(msgHeight.GetRevisionHeight()-lfHeight.GetRevisionHeight())
			}
			if err := wait(ctx, waitTime); err != nil {
				return err
			}
			return fmt.Errorf("msg(id=%v) not finalied: msg.height(%v) > lfh.height(%v)", msgID, msgHeight, lfHeight)
		}

		return nil
	}, rtyAtt, rtyDel, rtyErr, retry.Context(ctx)); err != nil {
		return nil, err
	}

	return msgRes, nil
}
