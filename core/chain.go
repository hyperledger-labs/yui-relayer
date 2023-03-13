package core

import (
	"context"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	transfertypes "github.com/cosmos/ibc-go/v4/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v4/modules/core/02-client/types"
	conntypes "github.com/cosmos/ibc-go/v4/modules/core/03-connection/types"
	chantypes "github.com/cosmos/ibc-go/v4/modules/core/04-channel/types"
	ibcexported "github.com/cosmos/ibc-go/v4/modules/core/exported"
)

// ProvableChain represents a chain that is supported by the relayer
type ProvableChain struct {
	Chain
	Prover
}

// NewProvableChain returns a new ProvableChain instance
func NewProvableChain(chain Chain, prover Prover) *ProvableChain {
	return &ProvableChain{Chain: chain, Prover: prover}
}

func (pc *ProvableChain) Init(homePath string, timeout time.Duration, codec codec.ProtoCodecMarshaler, debug bool) error {
	if err := pc.Chain.Init(homePath, timeout, codec, debug); err != nil {
		return err
	}
	if err := pc.Prover.Init(homePath, timeout, codec, debug); err != nil {
		return err
	}
	return nil
}

func (pc *ProvableChain) SetRelayInfo(path *PathEnd, counterparty *ProvableChain, counterpartyPath *PathEnd) error {
	if err := pc.Chain.SetRelayInfo(path, counterparty, counterpartyPath); err != nil {
		return err
	}
	if err := pc.Prover.SetRelayInfo(path, counterparty, counterpartyPath); err != nil {
		return err
	}
	return nil
}

func (pc *ProvableChain) SetupForRelay(ctx context.Context) error {
	if err := pc.Chain.SetupForRelay(ctx); err != nil {
		return err
	}
	if err := pc.Prover.SetupForRelay(ctx); err != nil {
		return err
	}
	return nil
}

// Chain represents a chain that supports sending transactions and querying the state
type Chain interface {
	// GetAddress returns the address of relayer
	GetAddress() (sdk.AccAddress, error)

	// Codec returns the codec
	Codec() codec.ProtoCodecMarshaler

	// Path returns the path
	Path() *PathEnd

	// Init initializes the chain
	Init(homePath string, timeout time.Duration, codec codec.ProtoCodecMarshaler, debug bool) error

	// SetRelayInfo sets source's path and counterparty's info to the chain
	SetRelayInfo(path *PathEnd, counterparty *ProvableChain, counterpartyPath *PathEnd) error

	// SetupForRelay performs chain-specific setup before starting the relay
	SetupForRelay(ctx context.Context) error

	// SendMsgs sends msgs to the chain
	SendMsgs(msgs []sdk.Msg) ([]byte, error)

	// Send sends msgs to the chain and logging a result of it
	// It returns a boolean value whether the result is success
	Send(msgs []sdk.Msg) bool

	// RegisterMsgEventListener registers a given EventListener to the chain
	RegisterMsgEventListener(MsgEventListener)

	ChainInfo
	IBCQuerier
	ICS20Querier
}

type ChainInfo interface {
	// ChainID returns ID of the chain
	ChainID() string

	// GetLatestHeight returns the latest height of the chain
	GetLatestHeight() (ibcexported.Height, error)
}

// MsgEventListener is a listener that listens a msg send to the chain
type MsgEventListener interface {
	// OnSentMsg is a callback functoin that is called when a msg send to the chain
	OnSentMsg(msgs []sdk.Msg) error
}

// IBCQuerier is an interface to the state of IBC
type IBCQuerier interface {
	ICS02Querier
	ICS03Querier
	ICS04Querier
}

// ICS02Querier is an interface to the state of ICS-02
type ICS02Querier interface {
	// QueryClientConsensusState retrevies the latest consensus state for a client in state at a given height
	QueryClientConsensusState(ctx QueryContext, dstClientConsHeight ibcexported.Height) (*clienttypes.QueryConsensusStateResponse, error)

	// QueryClientState returns the client state of dst chain
	// height represents the height of dst chain
	QueryClientState(ctx QueryContext) (*clienttypes.QueryClientStateResponse, error)
}

// ICS03Querier is an interface to the state of ICS-03
type ICS03Querier interface {
	// QueryConnection returns the remote end of a given connection
	QueryConnection(ctx QueryContext) (*conntypes.QueryConnectionResponse, error)
}

// ICS04Querier is an interface to the state of ICS-04
type ICS04Querier interface {
	// QueryChannel returns the channel associated with a channelID
	QueryChannel(ctx QueryContext) (chanRes *chantypes.QueryChannelResponse, err error)

	// QueryPacketCommitment returns the packet commitment corresponding to a given sequence
	QueryPacketCommitment(ctx QueryContext, seq uint64) (comRes *chantypes.QueryPacketCommitmentResponse, err error)

	// QueryPacketAcknowledgementCommitment returns the acknowledgement corresponding to a given sequence
	QueryPacketAcknowledgementCommitment(ctx QueryContext, seq uint64) (ackRes *chantypes.QueryPacketAcknowledgementResponse, err error)

	// QueryPacketCommitments returns an array of packet commitments
	QueryPacketCommitments(ctx QueryContext, offset, limit uint64) (comRes *chantypes.QueryPacketCommitmentsResponse, err error)

	// QueryUnrecievedPackets returns a list of unrelayed packet commitments
	QueryUnrecievedPackets(ctx QueryContext, seqs []uint64) ([]uint64, error)

	// QueryPacketAcknowledgementCommitments returns an array of packet acks
	QueryPacketAcknowledgementCommitments(ctx QueryContext, offset, limit uint64) (comRes *chantypes.QueryPacketAcknowledgementsResponse, err error)

	// QueryUnrecievedAcknowledgements returns a list of unrelayed packet acks
	QueryUnrecievedAcknowledgements(ctx QueryContext, seqs []uint64) ([]uint64, error)

	// QueryPacket returns the packet corresponding to a sequence
	QueryPacket(ctx QueryContext, sequence uint64) (*chantypes.Packet, error)

	// QueryPacketAcknowledgement returns the acknowledgement corresponding to a sequence
	QueryPacketAcknowledgement(ctx QueryContext, sequence uint64) ([]byte, error)
}

// ICS20Querier is an interface to the state of ICS20
type ICS20Querier interface {
	// QueryBalance returns the amount of coins in the relayer account
	QueryBalance(ctx QueryContext, address sdk.AccAddress) (sdk.Coins, error)

	// QueryDenomTraces returns all the denom traces from a given chain
	QueryDenomTraces(ctx QueryContext, offset, limit uint64) (*transfertypes.QueryDenomTracesResponse, error)
}

// QueryContext is a context that contains a height of the target chain for querying states
type QueryContext interface {
	// Context returns `context.Context``
	Context() context.Context

	// Height returns a height of the target chain for querying a state
	Height() ibcexported.Height
}

type queryContext struct {
	ctx    context.Context
	height ibcexported.Height
}

var _ QueryContext = (*queryContext)(nil)

// NewQueryContext returns a new context for querying states
func NewQueryContext(ctx context.Context, height ibcexported.Height) QueryContext {
	return queryContext{ctx: ctx, height: height}
}

// Context returns `context.Context``
func (qc queryContext) Context() context.Context {
	return qc.ctx
}

// Height returns a height of the target chain for querying a state
func (qc queryContext) Height() ibcexported.Height {
	return qc.height
}
