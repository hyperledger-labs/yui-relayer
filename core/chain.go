package core

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	clienttypes "github.com/cosmos/cosmos-sdk/x/ibc/core/02-client/types"
)

type ChainI interface {
	ClientType() string
	ChainID() string
	ClientID() string
	GetAddress() (sdk.AccAddress, error)

	QueryLatestHeader() (out HeaderI, err error)
	// height represents the height of src chain
	QueryClientState(height int64) (*clienttypes.QueryClientStateResponse, error)

	// Is first return value needed?
	SendMsgs(msgs []sdk.Msg) ([]byte, error)
	// Send sends msgs to the chain and logging a result of it
	// It returns a boolean value whether the result is success
	Send(msgs []sdk.Msg) bool

	// MakeMsgCreateClient creates a CreateClientMsg to this chain
	MakeMsgCreateClient(clientID string, dstHeader HeaderI, signer sdk.AccAddress) (sdk.Msg, error)

	StartEventListener(dst ChainI, strategy StrategyI)
}
