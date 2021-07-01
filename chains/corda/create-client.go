package corda

import (
	"context"

	"github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	clienttypes "github.com/cosmos/cosmos-sdk/x/ibc/core/02-client/types"
	cordatypes "github.com/hyperledger-labs/yui-corda-ibc/go/x/ibc/light-clients/xx-corda/types"
	"github.com/hyperledger-labs/yui-relayer/core"
	"google.golang.org/protobuf/types/known/emptypb"
)

// MakeMsgCreateClient creates a CreateClientMsg to this chain
func (c *Chain) MakeMsgCreateClient(clientID string, dstHeader core.HeaderI, signer sdk.AccAddress) (sdk.Msg, error) {
	// information for building consensus state can be obtained from host state
	host, err := c.client.host.QueryHost(
		context.TODO(),
		&emptypb.Empty{},
	)
	if err != nil {
		return nil, err
	}
	consensusState := cordatypes.ConsensusState{
		BaseId:    host.BaseId,
		NotaryKey: host.Notary.OwningKey,
	}

	// make client state
	clientState := cordatypes.ClientState{
		Id: clientID,
	}

	if anyClientState, err := types.NewAnyWithValue(&clientState); err != nil {
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
