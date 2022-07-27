package core

import (
	"context"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
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

// LightClientI is an interface to the light client
type LightClientI interface {
	// GetChainID returns the chain ID
	GetChainID() string

	// QueryLatestHeader returns the latest header from the chain
	QueryLatestHeader() (out HeaderI, err error)

	// GetLatestLightHeight returns the latest height on the light client
	GetLatestLightHeight() (int64, error)

	// CreateMsgCreateClient creates a CreateClientMsg to this chain
	CreateMsgCreateClient(clientID string, dstHeader HeaderI, signer sdk.AccAddress) (*clienttypes.MsgCreateClient, error)

	// SetupHeader creates a new header based on a given header
	SetupHeader(dst LightClientIBCQueryierI, baseSrcHeader HeaderI) (HeaderI, error)

	// UpdateLightWithHeader updates a header on the light client and returns the header and height corresponding to the chain
	UpdateLightWithHeader() (header HeaderI, provableHeight int64, queryableHeight int64, err error)
}

// LightClientIBCQueryierI is LightClientI + IBCQuerierI
type LightClientIBCQueryierI interface {
	LightClientI
	IBCQuerierI
}
