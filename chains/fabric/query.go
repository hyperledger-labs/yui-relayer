package fabric

import (
	"encoding/json"

	clienttypes "github.com/cosmos/cosmos-sdk/x/ibc/core/02-client/types"
	"github.com/datachainlab/fabric-ibc/commitment"
)

const (
	getSequenceFunc = "getSequence"
)

// QueryClientState retrevies the latest consensus state for a client in state at a given height
func (c *Chain) QueryClientState(height int64) (*clienttypes.QueryClientStateResponse, error) {
	panic("not implemented error")
}

// QueryCurrentSequence returns the current sequence for IBC chaincode
func (c *Chain) QueryCurrentSequence() (*commitment.Sequence, error) {
	res, err := c.Contract().EvaluateTransaction(getSequenceFunc)
	if err != nil {
		return nil, err
	}
	seq := new(commitment.Sequence)
	if err := json.Unmarshal(res, seq); err != nil {
		return nil, err
	}
	return seq, nil
}
