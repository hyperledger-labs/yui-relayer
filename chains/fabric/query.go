package fabric

import (
	clienttypes "github.com/cosmos/cosmos-sdk/x/ibc/core/02-client/types"
)

// QueryClientState retrevies the latest consensus state for a client in state at a given height
func (c *Chain) QueryClientState(height int64) (*clienttypes.QueryClientStateResponse, error) {
	panic("not implemented error")
}
