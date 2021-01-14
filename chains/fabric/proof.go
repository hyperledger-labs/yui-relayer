package fabric

import (
	"encoding/json"
	"fmt"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	conntypes "github.com/cosmos/cosmos-sdk/x/ibc/core/03-connection/types"
	chantypes "github.com/cosmos/cosmos-sdk/x/ibc/core/04-channel/types"
	"github.com/cosmos/cosmos-sdk/x/ibc/core/exported"
	"github.com/datachainlab/fabric-ibc/commitment"
	fabrictypes "github.com/datachainlab/fabric-ibc/x/ibc/light-clients/xx-fabric/types"
	"github.com/gogo/protobuf/proto"
)

const (
	endorseClientStateFunc     = "endorseClientState"
	endorseConsensusStateFunc  = "endorseConsensusStateCommitment"
	endorseConnectionStateFunc = "endorseConnectionState"
	endorseChannelStateFunc    = "endorseChannelState"
)

func (chain *Chain) endorseCommitment(fnName string, args []string, result proto.Message) (*fabrictypes.CommitmentProof, error) {
	fmt.Println("CALL ", fnName, args)
	txn, err := chain.Contract().CreateTransaction(fnName)
	if err != nil {
		return nil, err
	}
	res, err := txn.Simulate(args...)
	if err != nil {
		return nil, err
	}
	entry, err := unmarshalCommitmentEntry(res.Payload)
	if err != nil {
		return nil, err
	}
	if err = proto.Unmarshal(entry.Value, result); err != nil {
		return nil, err
	}
	return makeCommitmentProof(res)
}

func (chain *Chain) endorseClientState(clientID string) (exported.ClientState, *fabrictypes.CommitmentProof, error) {
	var (
		any    codectypes.Any
		result exported.ClientState
	)
	proof, err := chain.endorseCommitment(endorseClientStateFunc, []string{clientID}, &any)
	if err != nil {
		return nil, nil, err
	}
	if err := chain.Marshaler().UnpackAny(&any, &result); err != nil {
		return nil, nil, err
	}
	return result, proof, nil
}

func (chain *Chain) endorseConsensusState(clientID string, height uint64) (exported.ConsensusState, *fabrictypes.CommitmentProof, error) {
	var (
		any    codectypes.Any
		result exported.ConsensusState
	)
	proof, err := chain.endorseCommitment(endorseConsensusStateFunc, []string{clientID, fmt.Sprint(height)}, &any)
	if err != nil {
		return nil, nil, err
	}
	if err := chain.Marshaler().UnpackAny(&any, &result); err != nil {
		return nil, nil, err
	}
	return result, proof, nil
}

func (chain *Chain) endorseConnectionState(connectionID string) (*conntypes.ConnectionEnd, *fabrictypes.CommitmentProof, error) {
	var result conntypes.ConnectionEnd
	proof, err := chain.endorseCommitment(endorseConnectionStateFunc, []string{connectionID}, &result)
	if err != nil {
		return nil, nil, err
	}
	return &result, proof, nil
}

func (chain *Chain) endorseChannelState(portID, channelID string) (*chantypes.Channel, *fabrictypes.CommitmentProof, error) {
	var result chantypes.Channel
	proof, err := chain.endorseCommitment(endorseChannelStateFunc, []string{portID, channelID}, &result)
	if err != nil {
		return nil, nil, err
	}
	return &result, proof, nil
}

func unmarshalCommitmentEntry(payload []byte) (*commitment.Entry, error) {
	entry := new(commitment.Entry)
	if err := json.Unmarshal(payload, entry); err != nil {
		return nil, err
	}

	return entry, nil
}
