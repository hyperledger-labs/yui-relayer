package core

import (
	"context"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
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
	ProveState(ctx QueryContext, path string, value []byte) (proof []byte, proofHeight clienttypes.Height, err error)
}

// LightClient provides functions for creating and updating on-chain light clients on the counterparty chain
type LightClient interface {
	FinalityAware

	// CreateMsgCreateClient creates a MsgCreateClient for the counterparty chain
	CreateMsgCreateClient(selfHeader Header, signer sdk.AccAddress) (*clienttypes.MsgCreateClient, error)

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
	// GetFinalizedHeader returns the finalized header on this chain corresponding to `height`.
	// If `height` is nil, this function returns the latest finalized header.
	// If the header at `height` isn't finalized yet, this function returns an error.
	GetFinalizedHeader(height *uint64) (Header, error)
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
