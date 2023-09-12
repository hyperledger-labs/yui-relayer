package core

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/types"
)

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
