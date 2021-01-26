package corda

import sdk "github.com/cosmos/cosmos-sdk/types"

func (c *Chain) SendMsgs(msgs []sdk.Msg) ([]byte, error) {
	panic("not implemented error")
}

// Send sends msgs to the chain and logging a result of it
// It returns a boolean value whether the result is success
func (c *Chain) Send(msgs []sdk.Msg) bool {
	panic("not implemented error")
}
