package fabric

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	fabrictypes "github.com/hyperledger-labs/yui-fabric-ibc/x/ibc/light-clients/xx-fabric/types"
	"github.com/hyperledger-labs/yui-relayer/core"
)

// CreateMsgCreateClient creates a CreateClientMsg to this chain
func (pr *Prover) CreateMsgCreateClient(clientID string, dstHeader core.HeaderI, signer sdk.AccAddress) (*clienttypes.MsgCreateClient, error) {
	h := dstHeader.(*fabrictypes.Header)

	mspInfos, err := createMSPInitialClientState(h.MSPHeaders.Headers)
	if err != nil {
		return nil, err
	}
	clientState := &fabrictypes.ClientState{
		Id:                  clientID,
		LastChaincodeHeader: *h.ChaincodeHeader,
		LastChaincodeInfo:   *h.ChaincodeInfo,
		LastMspInfos:        *mspInfos,
	}
	consensusState := &fabrictypes.ConsensusState{
		Timestamp: h.ChaincodeHeader.Sequence.Timestamp,
	}

	return clienttypes.NewMsgCreateClient(
		clientState,
		consensusState,
		signer.String(),
	)
}

func createMSPInitialClientState(headers []fabrictypes.MSPHeader) (*fabrictypes.MSPInfos, error) {
	var infos fabrictypes.MSPInfos
	for _, mh := range headers {
		if mh.Type != fabrictypes.MSPHeaderTypeCreate {
			return nil, fmt.Errorf("unexpected fabric type: %v", mh.Type)
		}
		infos.Infos = append(infos.Infos, fabrictypes.MSPInfo{
			MSPID:   mh.MSPID,
			Config:  mh.Config,
			Policy:  mh.Policy,
			Freezed: false,
		})
	}
	return &infos, nil
}
