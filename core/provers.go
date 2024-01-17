package core

import (
	"context"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
)

// Prover represents a prover that supports generating a commitment proof
type Prover interface {
	// Init initializes the chain
	Init(homePath string, timeout time.Duration, codec codec.ProtoCodecMarshaler, debug bool) error

	// SetRelayInfo sets source's path and counterparty's info to the chain
	SetRelayInfo(path *PathEnd, counterparty *ProvableChain, counterpartyPath *PathEnd) error

	// SetupForRelay performs chain-specific setup before starting the relay
	SetupForRelay(ctx context.Context) error

	LightClient
	StateProver
}

// StateProver provides a generic way to generate existence proof of IBC states (e.g. ClientState, Connection, PacketCommitment, etc.)
type StateProver interface {
	// ProveState returns a proof of an IBC state specified by `path` and `value`
	ProveState(ctx QueryContext, path string, value []byte) (proof []byte, proofHeight clienttypes.Height, err error)

	// ProveHostConsensusState returns an existence proof of the consensus state at `height`
	// This proof would be ignored in ibc-go, but it is required to `getSelfConsensusState` of ibc-solidity.
	ProveHostConsensusState(ctx QueryContext, height exported.Height, consensusState exported.ConsensusState) (proof []byte, err error)
}

// LightClient provides functions for creating and updating on-chain light clients on the counterparty chain
type LightClient interface {
	FinalityAware

	// CreateInitialLightClientState returns a pair of ClientState and ConsensusState based on the state of the self chain at `height`.
	// These states will be submitted to the counterparty chain as MsgCreateClient.
	// If `height` is nil, the latest finalized height is selected automatically.
	CreateInitialLightClientState(height exported.Height) (exported.ClientState, exported.ConsensusState, error)

	// SetupHeadersForUpdate returns the finalized header and any intermediate headers needed to apply it to the client on the counterpaty chain
	// The order of the returned header slice should be as: [<intermediate headers>..., <update header>]
	// if the header slice's length == 0 and err == nil, the relayer should skips the update-client
	SetupHeadersForUpdate(counterparty FinalityAwareChain, latestFinalizedHeader Header) ([]Header, error)

	// CheckRefreshRequired returns if the on-chain light client needs to be updated.
	// For example, this requirement arises due to the trusting period mechanism.
	CheckRefreshRequired(counterparty ChainInfoICS02Querier) (bool, error)
}

// FinalityAware provides the capability to determine the finality of the chain
type FinalityAware interface {
	// GetLatestFinalizedHeader returns the latest finalized header on this chain
	// The returned header is expected to be the latest one of headers that can be verified by the light client
	GetLatestFinalizedHeader() (latestFinalizedHeader Header, err error)
}

// FinalityAwareChain is FinalityAware + Chain
type FinalityAwareChain interface {
	FinalityAware
	Chain
}

// ChainInfoICS02Querier is ChainInfo + ICS02Querier
type ChainInfoICS02Querier interface {
	ChainInfo
	ICS02Querier
}
