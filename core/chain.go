package core

import (
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	clienttypes "github.com/cosmos/cosmos-sdk/x/ibc/core/02-client/types"
	conntypes "github.com/cosmos/cosmos-sdk/x/ibc/core/03-connection/types"
	chantypes "github.com/cosmos/cosmos-sdk/x/ibc/core/04-channel/types"
	ibcexported "github.com/cosmos/cosmos-sdk/x/ibc/core/exported"
	"github.com/gogo/protobuf/proto"
)

type ChainI interface {
	ClientType() string
	ChainID() string
	ClientID() string

	GetAddress() (sdk.AccAddress, error)
	// TODO consider whether the name is appropriate.
	// GetLatestLightHeight uses the CLI utilities to pull the latest height from a given chain
	GetLatestLightHeight() (int64, error)
	Marshaler() codec.Marshaler

	SetPath(p *PathEnd) error
	Path() *PathEnd

	// QueryLatestHeight queries the chain for the latest height and returns it
	QueryLatestHeight() (int64, error)
	// QueryLatestHeader returns the latest header from the chain
	QueryLatestHeader() (out HeaderI, err error)
	// QueryClientConsensusState retrevies the latest consensus state for a client in state at a given height
	QueryClientConsensusState(height int64, dstClientConsHeight ibcexported.Height, prove bool) (*clienttypes.QueryConsensusStateResponse, error)
	// height represents the height of src chain
	QueryClientState(height int64, prove bool) (*clienttypes.QueryClientStateResponse, error)
	// QueryConnection returns the remote end of a given connection
	QueryConnection(height int64, prove bool) (*conntypes.QueryConnectionResponse, error)
	// QueryChannel returns the channel associated with a channelID
	QueryChannel(height int64, prove bool) (chanRes *chantypes.QueryChannelResponse, err error)

	SendMsgs(msgs []sdk.Msg) ([]byte, error)
	// Send sends msgs to the chain and logging a result of it
	// It returns a boolean value whether the result is success
	Send(msgs []sdk.Msg) bool

	Update(key, value string) (ChainConfigI, error)

	// MakeMsgCreateClient creates a CreateClientMsg to this chain
	MakeMsgCreateClient(clientID string, dstHeader HeaderI, signer sdk.AccAddress) (sdk.Msg, error)

	// CreateTrustedHeader creates ...
	CreateTrustedHeader(dstChain ChainI, srcHeader HeaderI) (HeaderI, error)

	StartEventListener(dst ChainI, strategy StrategyI)

	Init(homePath string, timeout time.Duration, debug bool) error
}

type ChainConfigI interface {
	proto.Message
	GetChain() ChainI
}
