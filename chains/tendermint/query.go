package tendermint

import (
	clienttypes "github.com/cosmos/cosmos-sdk/x/ibc/core/02-client/types"
)

// QueryLatestHeight queries the chain for the latest height and returns it
func (c *Chain) QueryLatestHeight() (int64, error) {
	return c.base.QueryLatestHeight()
}

// QueryClientState retrevies the latest consensus state for a client in state at a given height
func (c *Chain) QueryClientState(height int64) (*clienttypes.QueryClientStateResponse, error) {
	return c.base.QueryClientState(height)
}
