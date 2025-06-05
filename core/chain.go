package core

import (
	"context"
	"slices"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	conntypes "github.com/cosmos/ibc-go/v8/modules/core/03-connection/types"
	chantypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"
	"github.com/hyperledger-labs/yui-relayer/log"
	"github.com/hyperledger-labs/yui-relayer/otelcore/semconv"
	"go.opentelemetry.io/otel/trace"
)

//go:generate mockgen -source=chain.go -destination=mock_chain_test.go -package core
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

	// SendMsgs sends msgs to the chain and waits for them to be included in blocks.
	// This function returns err=nil only if all the msgs executed successfully at the blocks.
	// It should be noted that the block is not finalized at that point and can be reverted afterwards.
	SendMsgs(ctx context.Context, msgs []sdk.Msg) ([]MsgID, error)

	// GetMsgResult returns the execution result of `sdk.Msg` specified by `MsgID`
	// If the msg is not included in any block, this function waits for inclusion.
	GetMsgResult(ctx context.Context, id MsgID) (MsgResult, error)

	// RegisterMsgEventListener registers a given EventListener to the chain
	RegisterMsgEventListener(MsgEventListener)

	ChainInfo
	IBCQuerier
	ICS20Querier
}

// ChainInfo is an interface to the chain's general information
type ChainInfo interface {
	// ChainID returns ID of the chain
	ChainID() string

	// LatestHeight returns the latest height of the chain
	//
	// NOTE: The returned height does not have to be finalized.
	// If a finalized height/header is required, the `Prover`'s `GetLatestFinalizedHeader` function should be called instead.
	LatestHeight(ctx context.Context) (ibcexported.Height, error)

	// Timestamp returns the block timestamp
	Timestamp(ctx context.Context, height ibcexported.Height) (time.Time, error)

	// AverageBlockTime returns the average time required for each new block to be committed
	AverageBlockTime() time.Duration
}

// MsgEventListener is a listener that listens a msg send to the chain
type MsgEventListener interface {
	// OnSentMsg is a callback functoin that is called when a msg send to the chain
	OnSentMsg(ctx context.Context, msgs []sdk.Msg) error
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
	QueryConnection(ctx QueryContext, connectionID string) (*conntypes.QueryConnectionResponse, error)
}

// ICS04Querier is an interface to the state of ICS-04
type ICS04Querier interface {
	// QueryChannel returns the channel associated with a channelID
	QueryChannel(ctx QueryContext) (chanRes *chantypes.QueryChannelResponse, err error)

	// QueryNextSequenceReceive returns a info about nextSequence
	QueryNextSequenceReceive(ctx QueryContext) (res *chantypes.QueryNextSequenceReceiveResponse, err error)

	// QueryUnreceivedPackets returns a list of unrelayed packet commitments
	QueryUnreceivedPackets(ctx QueryContext, seqs []uint64) ([]uint64, error)

	// QueryUnfinalizedRelayedPackets returns packets and heights that are sent but not received at the latest finalized block on the counterparty chain
	QueryUnfinalizedRelayPackets(ctx QueryContext, counterparty LightClientICS04Querier) (PacketInfoList, error)

	// QueryUnreceivedAcknowledgements returns a list of unrelayed packet acks
	QueryUnreceivedAcknowledgements(ctx QueryContext, seqs []uint64) ([]uint64, error)

	// QueryUnfinalizedRelayedAcknowledgements returns acks and heights that are sent but not received at the latest finalized block on the counterpartychain
	QueryUnfinalizedRelayAcknowledgements(ctx QueryContext, counterparty LightClientICS04Querier) (PacketInfoList, error)

	// QueryChannelUpgrade returns the channel upgrade associated with a channelID
	QueryChannelUpgrade(ctx QueryContext) (*chantypes.QueryUpgradeResponse, error)

	// QueryChannelUpgradeError returns the channel upgrade error receipt associated with a channelID at the height of `ctx`.
	// WARN: This error receipt may not be used to cancel upgrade in FLUSHCOMPLETE state because of upgrade sequence mismatch.
	QueryChannelUpgradeError(ctx QueryContext) (*chantypes.QueryUpgradeErrorResponse, error)

	// QueryCanTransitionToFlushComplete returns the channel can transition to FLUSHCOMPLETE state.
	// Basically it requires that there remains no inflight packets.
	// Maybe additional condition for transition is required by the IBC/APP module.
	QueryCanTransitionToFlushComplete(ctx QueryContext) (bool, error)
}

// ICS20Querier is an interface to the state of ICS-20
type ICS20Querier interface {
	// QueryBalance returns the amount of coins in the relayer account
	QueryBalance(ctx QueryContext, address sdk.AccAddress) (sdk.Coins, error)

	// QueryDenomTraces returns all the denom traces from a given chain
	QueryDenomTraces(ctx QueryContext, offset, limit uint64) (*transfertypes.QueryDenomTracesResponse, error)
}

type LightClientICS04Querier interface {
	LightClient
	ICS04Querier
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

// Context returns `context.Context`
func (qc queryContext) Context() context.Context {
	return qc.ctx
}

// Height returns a height of the target chain for querying a state
func (qc queryContext) Height() ibcexported.Height {
	return qc.height
}

func GetChainLogger(chain ChainInfo) *log.RelayLogger {
	return log.GetLogger().
		WithChain(
			chain.ChainID(),
		).
		WithModule("core.chain")
}

func GetChainPairLogger(src, dst ChainInfo) *log.RelayLogger {
	return log.GetLogger().
		WithChainPair(
			src.ChainID(),
			dst.ChainID(),
		).
		WithModule("core.chain")
}

func WithChainAttributes(chainID string) trace.SpanStartOption {
	return trace.WithAttributes(
		semconv.ChainIDKey.String(chainID),
	)
}

func WithChainPairAttributes(src, dst ChainInfoLightClient) trace.SpanStartOption {
	return trace.WithAttributes(slices.Concat(
		semconv.AttributeGroup("src", semconv.ChainIDKey.String(src.ChainID())),
		semconv.AttributeGroup("dst", semconv.ChainIDKey.String(dst.ChainID())),
	)...)
}
