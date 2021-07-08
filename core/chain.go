package core

import (
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	transfertypes "github.com/cosmos/ibc-go/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	conntypes "github.com/cosmos/ibc-go/modules/core/03-connection/types"
	chantypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
	ibcexported "github.com/cosmos/ibc-go/modules/core/exported"
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
	Marshaler() codec.Codec

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
	// QueryBalance returns the amount of coins in the relayer account
	QueryBalance(address sdk.AccAddress) (sdk.Coins, error)
	// QueryDenomTraces returns all the denom traces from a given chain
	QueryDenomTraces(offset, limit uint64, height int64) (*transfertypes.QueryDenomTracesResponse, error)
	// QueryPacketCommitment returns the packet commitment proof at a given height
	QueryPacketCommitment(height int64, seq uint64) (comRes *chantypes.QueryPacketCommitmentResponse, err error)
	// QueryPacketCommitments returns an array of packet commitments
	QueryPacketCommitments(offset, limit, height uint64) (comRes *chantypes.QueryPacketCommitmentsResponse, err error)
	// QueryUnrecievedPackets returns a list of unrelayed packet commitments
	QueryUnrecievedPackets(height uint64, seqs []uint64) ([]uint64, error)
	// QueryPacketAcknowledgements returns an array of packet acks
	QueryPacketAcknowledgements(offset, limit, height uint64) (comRes *chantypes.QueryPacketAcknowledgementsResponse, err error)
	// QueryUnrecievedAcknowledgements returns a list of unrelayed packet acks
	QueryUnrecievedAcknowledgements(height uint64, seqs []uint64) ([]uint64, error)
	// QueryPacketAcknowledgementCommitment returns the packet ack proof at a given height
	QueryPacketAcknowledgementCommitment(height int64, seq uint64) (ackRes *chantypes.QueryPacketAcknowledgementResponse, err error)

	// QueryPacket returns a packet corresponds to a given sequence
	QueryPacket(height int64, sequence uint64) (*chantypes.Packet, error)
	QueryPacketAcknowledgement(height int64, sequence uint64) ([]byte, error)

	SendMsgs(msgs []sdk.Msg) ([]byte, error)
	// Send sends msgs to the chain and logging a result of it
	// It returns a boolean value whether the result is success
	Send(msgs []sdk.Msg) bool

	Update(key, value string) (ChainConfigI, error)

	// MakeMsgCreateClient creates a CreateClientMsg to this chain
	MakeMsgCreateClient(clientID string, dstHeader HeaderI, signer sdk.AccAddress) (sdk.Msg, error)

	// CreateTrustedHeader creates ...
	CreateTrustedHeader(dstChain ChainI, srcHeader HeaderI) (HeaderI, error)
	UpdateLightWithHeader() (HeaderI, error)

	StartEventListener(dst ChainI, strategy StrategyI)

	Init(homePath string, timeout time.Duration, debug bool) error
}

type ChainConfigI interface {
	proto.Message
	GetChain() ChainI
}
