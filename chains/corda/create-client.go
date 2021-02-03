package corda

import (
	"context"

	"github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	clienttypes "github.com/cosmos/cosmos-sdk/x/ibc/core/02-client/types"
	"github.com/datachainlab/relayer/core"
	"github.com/datachainlab/relayer/jp_datachain_corda_ibc_grpc"
)

// MakeMsgCreateClient creates a CreateClientMsg to this chain
func (c *Chain) MakeMsgCreateClient(clientID string, dstHeader core.HeaderI, signer sdk.AccAddress) (sdk.Msg, error) {
	// information for building consensus state can be obtained from host state
	host, err := c.client.hostAndBankQuery.QueryHost(
		context.TODO(),
		&jp_datachain_corda_ibc_grpc.QueryHostRequest{},
	)
	if err != nil {
		return nil, err
	}
	consensusState := jp_datachain_corda_ibc_grpc.ConsensusState{
		BaseId:    host.BaseId,
		NotaryKey: host.Notary.OwningKey,
	}

	if anyClientState, err := types.NewAnyWithValue(&jp_datachain_corda_ibc_grpc.ClientState{}); err != nil {
		return nil, err
	} else if anyConsensusState, err := types.NewAnyWithValue(&consensusState); err != nil {
		return nil, err
	} else {
		return &clienttypes.MsgCreateClient{
			ClientId:       clientID,
			ClientState:    anyClientState,
			ConsensusState: anyConsensusState,
			Signer:         signer.String(),
		}, nil
	}
}
