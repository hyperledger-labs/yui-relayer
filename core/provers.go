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
	// CreateMsgCreateClient creates a CreateClientMsg to this chain
	CreateMsgCreateClient(clientID string, dstHeader Header, signer sdk.AccAddress) (*clienttypes.MsgCreateClient, error)

	// GetLatestFinalizedHeader returns the latest finalized header on this chain
	// The returned header is expected to be the latest one of headers that can be verified by the light client
	GetLatestFinalizedHeader() (latestFinalizedHeader Header, err error)

	// SetupHeadersForUpdate returns the finalized header and any intermediate headers needed to apply it to the client on the counterpaty chain
	// The order of the returned header slice should be as: [<intermediate headers>..., <update header>]
	// if the header slice's length == 0 and err == nil, the relayer should skips the update-client
	SetupHeadersForUpdate(dstChain ChainInfoICS02Querier, latestFinalizedHeader Header) ([]Header, error)

	// CheckRefreshRequired returns if the on-chain light client needs to be updated.
	// For example, this requirement arises due to the trusting period mechanism.
	CheckRefreshRequired(counterparty Chain) (bool, error)
}

// ChainInfoICS02Querier is ChainInfo + ICS02Querier
type ChainInfoICS02Querier interface {
	ChainInfo
	ICS02Querier
}
