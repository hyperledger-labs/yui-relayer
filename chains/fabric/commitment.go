package fabric

import (
	fabrictypes "github.com/hyperledger-labs/yui-fabric-ibc/x/ibc/light-clients/xx-fabric/types"
	protocommon "github.com/hyperledger/fabric-protos-go/common"
	msppb "github.com/hyperledger/fabric-protos-go/msp"
	"github.com/hyperledger/fabric-protos-go/peer"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/hyperledger/fabric/common/policydsl"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/rwsetutil"
	"github.com/hyperledger/fabric/protoutil"

	"github.com/gogo/protobuf/proto"
)

func makeCommitmentProof(response *channel.Response) (*fabrictypes.CommitmentProof, error) {
	proof := &fabrictypes.CommitmentProof{}
	proof.Proposal = response.Responses[0].Payload

	for _, res := range response.Responses {
		proof.Signatures = append(proof.Signatures, res.GetEndorsement().GetSignature())
		proof.Identities = append(proof.Identities, res.GetEndorsement().GetEndorser())
	}

	proof.NsIndex = 1       // TODO?
	proof.WriteSetIndex = 0 // TODO?

	if err := proof.ValidateBasic(); err != nil {
		return nil, err
	}

	return proof, nil
}

func makeEndorsementPolicy(mspids []string) ([]byte, error) {
	return protoutil.Marshal(&protocommon.ApplicationPolicy{
		Type: &protocommon.ApplicationPolicy_SignaturePolicy{
			SignaturePolicy: policydsl.SignedByNOutOfGivenRole(int32(len(mspids)/2+1), msppb.MSPRole_PEER, mspids),
		},
	})
}

func makeIBCPolicy(mspids []string) ([]byte, error) {
	return protoutil.Marshal(&protocommon.ApplicationPolicy{
		Type: &protocommon.ApplicationPolicy_SignaturePolicy{
			SignaturePolicy: policydsl.SignedByNOutOfGivenRole(int32(len(mspids)/2+1), msppb.MSPRole_ADMIN, mspids),
		},
	})
}

func getTxReadWriteSetFromProposalResponsePayload(proposal []byte) (*peer.ChaincodeID, *rwsetutil.TxRwSet, error) {
	var payload peer.ProposalResponsePayload
	if err := proto.Unmarshal(proposal, &payload); err != nil {
		return nil, nil, err
	}
	var cact peer.ChaincodeAction
	if err := proto.Unmarshal(payload.Extension, &cact); err != nil {
		return nil, nil, err
	}
	txRWSet := &rwsetutil.TxRwSet{}
	if err := txRWSet.FromProtoBytes(cact.Results); err != nil {
		return nil, nil, err
	}
	return cact.GetChaincodeId(), txRWSet, nil
}
