package core

import (
	"context"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	conntypes "github.com/cosmos/ibc-go/v8/modules/core/03-connection/types"
	chantypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"
	"github.com/hyperledger-labs/yui-relayer/log"
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

	// SendMsgs sends msgs to the chain and waits for them to be included in blocks.
	// This function returns err=nil only if all the msgs executed successfully at the blocks.
	// It should be noted that the block is not finalized at that point and can be reverted afterwards.
	SendMsgs(msgs []sdk.Msg) ([]MsgID, error)

	// GetMsgResult returns the execution result of `sdk.Msg` specified by `MsgID`
	// If the msg is not included in any block, this function waits for inclusion.
	GetMsgResult(id MsgID) (MsgResult, error)

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
	LatestHeight() (ibcexported.Height, error)

	// Timestamp returns the block timestamp
	Timestamp(ibcexported.Height) (time.Time, error)

	// AverageBlockTime returns the average time required for each new block to be committed
	AverageBlockTime() time.Duration
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
	QueryConnection(ctx QueryContext, connectionID string) (*conntypes.QueryConnectionResponse, error)
}

// ICS04Querier is an interface to the state of ICS-04
type ICS04Querier interface {
	// QueryChannel returns the channel associated with a channelID
	QueryChannel(ctx QueryContext) (chanRes *chantypes.QueryChannelResponse, err error)

	// QueryUnreceivedPackets returns a list of unrelayed packet commitments
	QueryUnreceivedPackets(ctx QueryContext, seqs []uint64) ([]uint64, error)

	// QueryUnfinalizedRelayedPackets returns packets and heights that are sent but not received at the latest finalized block on the counterparty chain
	QueryUnfinalizedRelayPackets(ctx QueryContext, counterparty LightClientICS04Querier) (PacketInfoList, error)

	// QueryUnreceivedAcknowledgements returns a list of unrelayed packet acks
	QueryUnreceivedAcknowledgements(ctx QueryContext, seqs []uint64) ([]uint64, error)

	// QueryUnfinalizedRelayedAcknowledgements returns acks and heights that are sent but not received at the latest finalized block on the counterpartychain
	QueryUnfinalizedRelayAcknowledgements(ctx QueryContext, counterparty LightClientICS04Querier) (PacketInfoList, error)
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
