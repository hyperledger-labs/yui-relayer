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
)

// ProvableChain represents a chain that is supported by the relayer
type ProvableChain struct {
	ChainI
	ProverI
}

// NewProvableChain returns a new ProvableChain instance
func NewProvableChain(chain ChainI, prover ProverI) *ProvableChain {
	return &ProvableChain{ChainI: chain, ProverI: prover}
}

// ChainI represents a chain that supports sending transactions and querying the state
type ChainI interface {
	// ChainID returns ID of the chain
	ChainID() string

	// GetLatestHeight gets the chain for the latest height and returns it
	GetLatestHeight() (int64, error)

	// GetAddress returns the address of relayer
	GetAddress() (sdk.AccAddress, error)

	// Codec returns the codec
	Codec() codec.ProtoCodecMarshaler

	// SetPath sets a given path to the chain
	SetPath(p *PathEnd) error

	// Path returns the path
	Path() *PathEnd

	// SendMsgs sends msgs to the chain
	SendMsgs(msgs []sdk.Msg) ([]byte, error)

	// Send sends msgs to the chain and logging a result of it
	// It returns a boolean value whether the result is success
	Send(msgs []sdk.Msg) bool

	// StartEventListener ...
	StartEventListener(dst ChainI, strategy StrategyI)

	// Init ...
	Init(homePath string, timeout time.Duration, codec codec.ProtoCodecMarshaler, debug bool) error

	IBCQuerierI
}

// IBCQuerierI is an interface to the state of IBC
type IBCQuerierI interface {
	// QueryClientConsensusState retrevies the latest consensus state for a client in state at a given height
	QueryClientConsensusState(height int64, dstClientConsHeight ibcexported.Height) (*clienttypes.QueryConsensusStateResponse, error)

	// QueryClientState returns the client state of dst chain
	// height represents the height of dst chain
	QueryClientState(height int64) (*clienttypes.QueryClientStateResponse, error)

	// QueryConnection returns the remote end of a given connection
	QueryConnection(height int64) (*conntypes.QueryConnectionResponse, error)

	// QueryChannel returns the channel associated with a channelID
	QueryChannel(height int64) (chanRes *chantypes.QueryChannelResponse, err error)

	// QueryPacketCommitment returns the packet commitment corresponding to a given sequence
	QueryPacketCommitment(height int64, seq uint64) (comRes *chantypes.QueryPacketCommitmentResponse, err error)

	// QueryPacketAcknowledgementCommitment returns the acknowledgement corresponding to a given sequence
	QueryPacketAcknowledgementCommitment(height int64, seq uint64) (ackRes *chantypes.QueryPacketAcknowledgementResponse, err error)

	// QueryPacketCommitments returns an array of packet commitments
	QueryPacketCommitments(offset, limit uint64, height int64) (comRes *chantypes.QueryPacketCommitmentsResponse, err error)

	// QueryUnrecievedPackets returns a list of unrelayed packet commitments
	QueryUnrecievedPackets(height int64, seqs []uint64) ([]uint64, error)

	// QueryPacketAcknowledgementCommitments returns an array of packet acks
	QueryPacketAcknowledgementCommitments(offset, limit uint64, height int64) (comRes *chantypes.QueryPacketAcknowledgementsResponse, err error)

	// QueryUnrecievedAcknowledgements returns a list of unrelayed packet acks
	QueryUnrecievedAcknowledgements(height int64, seqs []uint64) ([]uint64, error)

	// QueryPacket returns the packet corresponding to a sequence
	QueryPacket(height int64, sequence uint64) (*chantypes.Packet, error)

	// QueryPacketAcknowledgement returns the acknowledgement corresponding to a sequence
	QueryPacketAcknowledgement(height int64, sequence uint64) ([]byte, error)

	// QueryBalance returns the amount of coins in the relayer account
	QueryBalance(address sdk.AccAddress) (sdk.Coins, error)

	// QueryDenomTraces returns all the denom traces from a given chain
	QueryDenomTraces(offset, limit uint64, height int64) (*transfertypes.QueryDenomTracesResponse, error)
}

// IBCProvableQuerierI is an interface to the state of IBC and its proof.
type IBCProvableQuerierI interface {
	// QueryClientConsensusState returns the ClientConsensusState and its proof
	QueryClientConsensusStateWithProof(height int64, dstClientConsHeight ibcexported.Height) (*clienttypes.QueryConsensusStateResponse, error)

	// QueryClientStateWithProof returns the ClientState and its proof
	QueryClientStateWithProof(height int64) (*clienttypes.QueryClientStateResponse, error)

	// QueryConnectionWithProof returns the Connection and its proof
	QueryConnectionWithProof(height int64) (*conntypes.QueryConnectionResponse, error)

	// QueryChannelWithProof returns the Channel and its proof
	QueryChannelWithProof(height int64) (chanRes *chantypes.QueryChannelResponse, err error)

	// QueryPacketCommitmentWithProof returns the packet commitment and its proof
	QueryPacketCommitmentWithProof(height int64, seq uint64) (comRes *chantypes.QueryPacketCommitmentResponse, err error)

	// QueryPacketAcknowledgementCommitmentWithProof returns the packet acknowledgement commitment and its proof
	QueryPacketAcknowledgementCommitmentWithProof(height int64, seq uint64) (ackRes *chantypes.QueryPacketAcknowledgementResponse, err error)
}
