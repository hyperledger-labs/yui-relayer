package fabric

import (
	"encoding/json"
	"fmt"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	conntypes "github.com/cosmos/ibc-go/modules/core/03-connection/types"
	chantypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
	"github.com/cosmos/ibc-go/modules/core/exported"
	"github.com/gogo/protobuf/proto"
	"github.com/hyperledger-labs/yui-fabric-ibc/commitment"
	fabrictypes "github.com/hyperledger-labs/yui-fabric-ibc/x/ibc/light-clients/xx-fabric/types"
)

const (
	endorseClientStateFunc           = "endorseClientState"
	endorseConsensusStateFunc        = "endorseConsensusStateCommitment"
	endorseConnectionStateFunc       = "endorseConnectionState"
	endorseChannelStateFunc          = "endorseChannelState"
	endorsePacketCommitmentFunc      = "endorsePacketCommitment"
	endorsePacketAcknowledgementFunc = "endorsePacketAcknowledgement"
)

func (pr *Prover) endorseCommitment(fnName string, args []string, result proto.Message) (*fabrictypes.CommitmentProof, error) {
	v, proof, err := pr.endorseCommitmentRaw(fnName, args)
	if err != nil {
		return nil, err
	}
	if err = proto.Unmarshal(v, result); err != nil {
		return nil, err
	}
	return proof, nil
}

func (pr *Prover) endorseCommitmentRaw(fnName string, args []string) ([]byte, *fabrictypes.CommitmentProof, error) {
	txn, err := pr.chain.Contract().CreateTransaction(fnName)
	if err != nil {
		return nil, nil, err
	}
	res, err := txn.Simulate(args...)
	if err != nil {
		return nil, nil, err
	}
	entry, err := unmarshalCommitmentEntry(res.Payload)
	if err != nil {
		return nil, nil, err
	}
	proof, err := makeCommitmentProof(res)
	if err != nil {
		return nil, nil, err
	}
	return entry.Value, proof, nil
}

func (pr *Prover) endorseClientState(clientID string) (exported.ClientState, *fabrictypes.CommitmentProof, error) {
	var (
		any    codectypes.Any
		result exported.ClientState
	)
	proof, err := pr.endorseCommitment(endorseClientStateFunc, []string{clientID}, &any)
	if err != nil {
		return nil, nil, err
	}
	if err := pr.chain.Codec().UnpackAny(&any, &result); err != nil {
		return nil, nil, err
	}
	return result, proof, nil
}

func (pr *Prover) endorseConsensusState(clientID string, height uint64) (exported.ConsensusState, *fabrictypes.CommitmentProof, error) {
	var (
		any    codectypes.Any
		result exported.ConsensusState
	)
	proof, err := pr.endorseCommitment(endorseConsensusStateFunc, []string{clientID, fmt.Sprint(height)}, &any)
	if err != nil {
		return nil, nil, err
	}
	if err := pr.chain.Codec().UnpackAny(&any, &result); err != nil {
		return nil, nil, err
	}
	return result, proof, nil
}

func (pr *Prover) endorseConnectionState(connectionID string) (*conntypes.ConnectionEnd, *fabrictypes.CommitmentProof, error) {
	var result conntypes.ConnectionEnd
	proof, err := pr.endorseCommitment(endorseConnectionStateFunc, []string{connectionID}, &result)
	if err != nil {
		return nil, nil, err
	}
	return &result, proof, nil
}

func (pr *Prover) endorseChannelState(portID, channelID string) (*chantypes.Channel, *fabrictypes.CommitmentProof, error) {
	var result chantypes.Channel
	proof, err := pr.endorseCommitment(endorseChannelStateFunc, []string{portID, channelID}, &result)
	if err != nil {
		return nil, nil, err
	}
	return &result, proof, nil
}

func (pr *Prover) endorsePacketCommitment(portID, channelID string, sequence uint64) ([]byte, *fabrictypes.CommitmentProof, error) {
	return pr.endorseCommitmentRaw(endorsePacketCommitmentFunc, []string{portID, channelID, fmt.Sprint(sequence)})
}

func (pr *Prover) endorsePacketAcknowledgement(portID, channelID string, sequence uint64) ([]byte, *fabrictypes.CommitmentProof, error) {
	return pr.endorseCommitmentRaw(endorsePacketAcknowledgementFunc, []string{portID, channelID, fmt.Sprint(sequence)})
}

func unmarshalCommitmentEntry(payload []byte) (*commitment.Entry, error) {
	entry := new(commitment.Entry)
	if err := json.Unmarshal(payload, entry); err != nil {
		return nil, err
	}

	return entry, nil
}
