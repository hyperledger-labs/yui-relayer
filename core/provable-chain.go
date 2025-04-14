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
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// ProvableChain represents a chain that is supported by the relayer.
//
// It wraps primary methods of the Chain and Prover interfaces with tracing.
// This allows the relayer to provide tracing functionality without modifying module code.
//
// Modules can also add custom attributes to spans. For example, a module can add attributes
// in the GetMsgResult method as follows:
//
//	func (c *Chain) GetMsgResult(ctx context.Context, id core.MsgID) (core.MsgResult, error) {
//		msgID, ok := id.(*MsgID)
//		if !ok {
//			return nil, fmt.Errorf("unexpected message id type: %T", id)
//		}
//
//		span := trace.SpanFromContext(ctx)
//		span.SetAttributes(core.AttributeKeyTxHash.String(msgID.TxHash))
//
//		// -- snip --
//	}
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

func (pc *ProvableChain) SendMsgs(ctx context.Context, msgs []sdk.Msg) ([]MsgID, error) {
	ctx, span := tracer.Start(ctx, "Chain.SendMsgs",
		WithChainAttributes(pc.ChainID()),
		withPackage(pc.Chain),
	)
	defer span.End()

	ids, err := pc.Chain.SendMsgs(ctx, msgs)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}
	return ids, err
}

func (pc *ProvableChain) GetMsgResult(ctx context.Context, id MsgID) (MsgResult, error) {
	ctx, span := tracer.Start(ctx, "Chain.GetMsgResult",
		WithChainAttributes(pc.ChainID()),
		withPackage(pc.Chain),
	)
	defer span.End()

	result, err := pc.Chain.GetMsgResult(ctx, id)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}
	return result, err
}

func (pc *ProvableChain) LatestHeight(ctx context.Context) (ibcexported.Height, error) {
	ctx, span := tracer.Start(ctx, "Chain.LatestHeight",
		WithChainAttributes(pc.ChainID()),
		withPackage(pc.Chain),
	)
	defer span.End()

	height, err := pc.Chain.LatestHeight(ctx)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}
	return height, err
}

func (pc *ProvableChain) Timestamp(ctx context.Context, height ibcexported.Height) (time.Time, error) {
	ctx, span := tracer.Start(ctx, "Chain.Timestamp",
		WithChainAttributes(pc.ChainID()),
		withPackage(pc.Chain),
	)
	defer span.End()

	t, err := pc.Chain.Timestamp(ctx, height)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}
	return t, err
}

func (pc *ProvableChain) QueryClientConsensusState(ctx QueryContext, dstClientConsHeight ibcexported.Height) (*clienttypes.QueryConsensusStateResponse, error) {
	ctx, span := StartTraceWithQueryContext(tracer, ctx, "Chain.QueryClientConsensusState",
		WithClientAttributes(pc),
		withPackage(pc.Chain),
	)
	defer span.End()

	resp, err := pc.Chain.QueryClientConsensusState(ctx, dstClientConsHeight)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}
	return resp, err
}

func (pc *ProvableChain) QueryClientState(ctx QueryContext) (*clienttypes.QueryClientStateResponse, error) {
	ctx, span := StartTraceWithQueryContext(tracer, ctx, "Chain.QueryClientState",
		WithChainAttributes(pc.ChainID()),
		withPackage(pc.Chain),
	)
	defer span.End()

	resp, err := pc.Chain.QueryClientState(ctx)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}
	return resp, err
}

func (pc *ProvableChain) QueryConnection(ctx QueryContext, connectionID string) (*conntypes.QueryConnectionResponse, error) {
	ctx, span := StartTraceWithQueryContext(tracer, ctx, "Chain.QueryConnection",
		WithClientAttributes(pc),
		trace.WithAttributes(AttributeKeyConnectionID.String(connectionID)),
		withPackage(pc.Chain),
	)
	defer span.End()

	resp, err := pc.Chain.QueryConnection(ctx, connectionID)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}
	return resp, err
}

func (pc *ProvableChain) QueryChannel(ctx QueryContext) (*chantypes.QueryChannelResponse, error) {
	ctx, span := StartTraceWithQueryContext(tracer, ctx, "Chain.QueryChannel",
		WithChannelAttributes(pc),
		withPackage(pc.Chain),
	)
	defer span.End()

	resp, err := pc.Chain.QueryChannel(ctx)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}
	return resp, err
}

func (pc *ProvableChain) QueryUnreceivedPackets(ctx QueryContext, seqs []uint64) ([]uint64, error) {
	ctx, span := StartTraceWithQueryContext(tracer, ctx, "Chain.QueryUnreceivedPackets",
		WithChannelAttributes(pc),
		withPackage(pc.Chain),
	)
	defer span.End()

	packets, err := pc.Chain.QueryUnreceivedPackets(ctx, seqs)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}
	return packets, err
}

func (pc *ProvableChain) QueryUnfinalizedRelayPackets(ctx QueryContext, counterparty LightClientICS04Querier) (PacketInfoList, error) {
	ctx, span := StartTraceWithQueryContext(tracer, ctx, "Chain.QueryUnfinalizedRelayPackets",
		WithChannelAttributes(pc),
		withPackage(pc.Chain),
	)
	defer span.End()

	list, err := pc.Chain.QueryUnfinalizedRelayPackets(ctx, counterparty)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}
	return list, err
}

func (pc *ProvableChain) QueryUnreceivedAcknowledgements(ctx QueryContext, seqs []uint64) ([]uint64, error) {
	ctx, span := StartTraceWithQueryContext(tracer, ctx, "Chain.QueryUnreceivedAcknowledgements",
		WithChannelAttributes(pc),
		withPackage(pc.Chain),
	)
	defer span.End()

	acks, err := pc.Chain.QueryUnreceivedAcknowledgements(ctx, seqs)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}
	return acks, err
}

func (pc *ProvableChain) QueryUnfinalizedRelayAcknowledgements(ctx QueryContext, counterparty LightClientICS04Querier) (PacketInfoList, error) {
	ctx, span := StartTraceWithQueryContext(tracer, ctx, "Chain.QueryUnfinalizedRelayAcknowledgements",
		WithChannelAttributes(pc),
		withPackage(pc.Chain),
	)
	defer span.End()

	list, err := pc.Chain.QueryUnfinalizedRelayAcknowledgements(ctx, counterparty)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}
	return list, err
}

func (pc *ProvableChain) QueryChannelUpgrade(ctx QueryContext) (*chantypes.QueryUpgradeResponse, error) {
	ctx, span := StartTraceWithQueryContext(tracer, ctx, "Chain.QueryChannelUpgrade",
		WithChannelAttributes(pc),
		withPackage(pc.Chain),
	)
	defer span.End()

	resp, err := pc.Chain.QueryChannelUpgrade(ctx)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}
	return resp, err
}

func (pc *ProvableChain) QueryChannelUpgradeError(ctx QueryContext) (*chantypes.QueryUpgradeErrorResponse, error) {
	ctx, span := StartTraceWithQueryContext(tracer, ctx, "Chain.QueryChannelUpgradeError",
		WithChannelAttributes(pc),
		withPackage(pc.Chain),
	)
	defer span.End()

	resp, err := pc.Chain.QueryChannelUpgradeError(ctx)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}
	return resp, err
}

func (pc *ProvableChain) QueryCanTransitionToFlushComplete(ctx QueryContext) (bool, error) {
	ctx, span := StartTraceWithQueryContext(tracer, ctx, "Chain.QueryCanTransitionToFlushComplete",
		WithChannelAttributes(pc),
		withPackage(pc.Chain),
	)
	defer span.End()

	ok, err := pc.Chain.QueryCanTransitionToFlushComplete(ctx)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}
	return ok, err
}

func (pc *ProvableChain) QueryBalance(ctx QueryContext, address sdk.AccAddress) (sdk.Coins, error) {
	ctx, span := StartTraceWithQueryContext(tracer, ctx, "Chain.QueryBalance",
		WithChainAttributes(pc.ChainID()),
		withPackage(pc.Chain),
	)
	defer span.End()

	coins, err := pc.Chain.QueryBalance(ctx, address)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}
	return coins, err
}

func (pc *ProvableChain) QueryDenomTraces(ctx QueryContext, offset, limit uint64) (*transfertypes.QueryDenomTracesResponse, error) {
	ctx, span := StartTraceWithQueryContext(tracer, ctx, "Chain.QueryDenomTraces",
		WithChainAttributes(pc.ChainID()),
		withPackage(pc.Chain),
	)
	defer span.End()

	resp, err := pc.Chain.QueryDenomTraces(ctx, offset, limit)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}
	return resp, err
}

func (pc *ProvableChain) GetLatestFinalizedHeader(ctx context.Context) (Header, error) {
	ctx, span := tracer.Start(ctx, "Prover.GetLatestFinalizedHeader",
		WithChainAttributes(pc.ChainID()),
		withPackage(pc.Prover),
	)
	defer span.End()

	header, err := pc.Prover.GetLatestFinalizedHeader(ctx)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}
	return header, err
}

func (pc *ProvableChain) CreateInitialLightClientState(ctx context.Context, height ibcexported.Height) (ibcexported.ClientState, ibcexported.ConsensusState, error) {
	ctx, span := tracer.Start(ctx, "Prover.CreateInitialLightClientState",
		WithChainAttributes(pc.ChainID()),
		withPackage(pc.Prover),
	)
	defer span.End()

	clientState, consensusState, err := pc.Prover.CreateInitialLightClientState(ctx, height)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}
	return clientState, consensusState, err
}

func (pc *ProvableChain) SetupHeadersForUpdate(ctx context.Context, counterparty FinalityAwareChain, latestFinalizedHeader Header) ([]Header, error) {
	ctx, span := tracer.Start(ctx, "Prover.SetupHeadersForUpdate",
		WithChainAttributes(pc.ChainID()),
		trace.WithAttributes(attribute.String("counterparty_chain_id", counterparty.ChainID())),
		withPackage(pc.Prover),
	)
	defer span.End()

	headers, err := pc.Prover.SetupHeadersForUpdate(ctx, counterparty, latestFinalizedHeader)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}
	return headers, err
}

func (pc *ProvableChain) CheckRefreshRequired(ctx context.Context, counterparty ChainInfoICS02Querier) (bool, error) {
	ctx, span := tracer.Start(ctx, "Prover.CheckRefreshRequired",
		WithChainAttributes(pc.ChainID()),
		trace.WithAttributes(attribute.String("counterparty_chain_id", counterparty.ChainID())),
		withPackage(pc.Prover),
	)
	defer span.End()

	required, err := pc.Prover.CheckRefreshRequired(ctx, counterparty)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}
	return required, err
}

func (pc *ProvableChain) ProveState(ctx QueryContext, path string, value []byte) ([]byte, clienttypes.Height, error) {
	ctx, span := StartTraceWithQueryContext(tracer, ctx, "Prover.ProveState",
		WithChainAttributes(pc.ChainID()),
		trace.WithAttributes(AttributeKeyPath.String(path)),
		withPackage(pc.Prover),
	)
	defer span.End()

	proof, height, err := pc.Prover.ProveState(ctx, path, value)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}
	return proof, height, err
}

func (pc *ProvableChain) ProveHostConsensusState(ctx QueryContext, height ibcexported.Height, consensusState ibcexported.ConsensusState) ([]byte, error) {
	ctx, span := StartTraceWithQueryContext(tracer, ctx, "Prover.ProveHostConsensusState",
		WithChainAttributes(pc.ChainID()),
		withPackage(pc.Prover),
	)
	defer span.End()

	proof, err := pc.Prover.ProveHostConsensusState(ctx, height, consensusState)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}
	return proof, err
}
