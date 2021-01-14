package tendermint

import (
	clienttypes "github.com/cosmos/cosmos-sdk/x/ibc/core/02-client/types"
	conntypes "github.com/cosmos/cosmos-sdk/x/ibc/core/03-connection/types"
	chantypes "github.com/cosmos/cosmos-sdk/x/ibc/core/04-channel/types"
	ibcexported "github.com/cosmos/cosmos-sdk/x/ibc/core/exported"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
)

// QueryLatestHeight queries the chain for the latest height and returns it
func (c *Chain) QueryLatestHeight() (int64, error) {
	return c.base.QueryLatestHeight()
}

// QueryValsetAtHeight returns the validator set at a given height
func (c *Chain) QueryValsetAtHeight(height clienttypes.Height) (*tmproto.ValidatorSet, error) {
	return c.base.QueryValsetAtHeight(height)
}

// QueryClientState retrevies the latest consensus state for a client in state at a given height
func (c *Chain) QueryClientState(height int64, prove bool) (*clienttypes.QueryClientStateResponse, error) {
	// TODO use arg `prove` to call the method
	return c.base.QueryClientState(height)
}

// QueryConnection returns the remote end of a given connection
func (c *Chain) QueryConnection(height int64, prove bool) (*conntypes.QueryConnectionResponse, error) {
	// TODO use arg `prove` to call the method
	return c.base.QueryConnection(height)
}

// QueryChannel returns the channel associated with a channelID
func (c *Chain) QueryChannel(height int64, prove bool) (*chantypes.QueryChannelResponse, error) {
	// TODO use arg `prove` to call the method
	return c.base.QueryChannel(height)
}

// QueryClientConsensusState retrevies the latest consensus state for a client in state at a given height
func (c *Chain) QueryClientConsensusState(height int64, dstClientConsHeight ibcexported.Height, prove bool) (*clienttypes.QueryConsensusStateResponse, error) {
	// TODO use arg `prove` to call the method
	return c.base.QueryClientConsensusState(height, dstClientConsHeight)
}
