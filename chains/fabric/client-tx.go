package fabric

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	clienttypes "github.com/cosmos/cosmos-sdk/x/ibc/core/02-client/types"
	fabrictypes "github.com/datachainlab/fabric-ibc/x/ibc/light-clients/xx-fabric/types"
	"github.com/datachainlab/relayer/core"
	"github.com/gogo/protobuf/proto"
)

func (dst *Chain) MakeMsgCreateClient(clientID string, dstHeader core.HeaderI, signer sdk.AccAddress) (sdk.Msg, error) {
	dstSeq, err := dst.QueryCurrentSequence()
	if err != nil {
		return nil, err
	}

	var ccid = fabrictypes.ChaincodeID{
		Name:    dst.config.ChaincodeId,
		Version: "1", // TODO add version to config
	}

	pcBytes, err := makeEndorsementPolicy(dst.config.EndorsementPolicies)
	if err != nil {
		return nil, err
	}
	ipBytes, err := makeIBCPolicy(dst.config.IbcPolicies)
	if err != nil {
		return nil, err
	}
	ci := fabrictypes.NewChaincodeInfo(dst.config.Channel, ccid, pcBytes, ipBytes, nil)
	ch := fabrictypes.NewChaincodeHeader(
		dstSeq.Value,
		dstSeq.Timestamp,
		fabrictypes.CommitmentProof{},
	)

	mspConfs, err := dst.GetLocalMspConfigs()
	if err != nil {
		return nil, err
	}
	hs := []fabrictypes.MSPHeader{}
	for _, mc := range mspConfs {
		mcBytes, err := proto.Marshal(&mc)
		if err != nil {
			return nil, err
		}
		hs = append(hs, fabrictypes.NewMSPHeader(fabrictypes.MSPHeaderTypeCreate, dst.config.MspId, mcBytes, ipBytes, &fabrictypes.MessageProof{}))
	}
	mhs := fabrictypes.NewMSPHeaders(hs)
	mspInfos, err := createMSPInitialClientState(mhs.Headers)
	if err != nil {
		return nil, err
	}
	clientState := &fabrictypes.ClientState{
		Id:                  clientID,
		LastChaincodeHeader: ch,
		LastChaincodeInfo:   ci,
		LastMspInfos:        *mspInfos,
	}
	consensusState := &fabrictypes.ConsensusState{
		Timestamp: dstSeq.Timestamp,
	}

	return clienttypes.NewMsgCreateClient(
		clientID,
		clientState,
		consensusState,
		signer,
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
