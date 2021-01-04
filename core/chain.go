package core

import sdk "github.com/cosmos/cosmos-sdk/types"

type ChainI interface {
	ClientType() string
	QueryLatestHeader() (out HeaderI, err error)
	// Is first return value needed?
	SendMsgs(msgs []sdk.Msg) ([]byte, error)
	// Send sends msgs to the chain and logging a result of it
	// It returns a boolean value whether the result is success
	Send(msgs []sdk.Msg) bool

	CreateClients(dst ChainI) (err error)

	StartEventListener(dst ChainI, strategy StrategyI)
}
