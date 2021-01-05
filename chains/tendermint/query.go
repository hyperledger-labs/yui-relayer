package tendermint

import (
	clientutils "github.com/cosmos/cosmos-sdk/x/ibc/core/02-client/client/utils"
	clienttypes "github.com/cosmos/cosmos-sdk/x/ibc/core/02-client/types"
)

// QueryClientState retrevies the latest consensus state for a client in state at a given height
func (c *Chain) QueryClientState(height int64) (*clienttypes.QueryClientStateResponse, error) {
	return clientutils.QueryClientStateABCI(c.base.CLIContext(height), c.base.PathEnd.ClientID)
}
