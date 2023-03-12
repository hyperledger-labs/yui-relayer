package core

import (
	"context"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	clienttypes "github.com/cosmos/ibc-go/v4/modules/core/02-client/types"
)

// ProverI represents a prover that supports generating a commitment proof
type ProverI interface {
	// Init initializes the chain
	Init(homePath string, timeout time.Duration, codec codec.ProtoCodecMarshaler, debug bool) error

	// SetRelayInfo sets source's path and counterparty's info to the chain
	SetRelayInfo(path *PathEnd, counterparty *ProvableChain, counterpartyPath *PathEnd) error

	// SetupForRelay performs chain-specific setup before starting the relay
	SetupForRelay(ctx context.Context) error

	LightClientI
	IBCProvableQuerierI
}

// LightClientI provides functions for creating and updating on-chain light clients on the counterparty chain
type LightClientI interface {
	// GetChainID returns the chain ID
	GetChainID() string

	// CreateMsgCreateClient creates a CreateClientMsg to this chain
	CreateMsgCreateClient(clientID string, dstHeader HeaderI, signer sdk.AccAddress) (*clienttypes.MsgCreateClient, error)

	// GetLatestFinalizedHeader returns the latest finalized header on this chain
	// The returned header is expected to be the latest one of headers that can be verified by the light client
	GetLatestFinalizedHeader() (latestFinalizedHeader HeaderI, err error)

	// SetupHeadersForUpdate returns the finalized header and any intermediate headers needed to apply it to the client on the counterpaty chain
	// The order of the returned header slice should be as: [<intermediate headers>..., <update header>]
	// if the header slice's length == nil and err == nil, the relayer should skips the update-client
	SetupHeadersForUpdate(dstChain ChainI, latestFinalizedHeader HeaderI) ([]HeaderI, error)
}
