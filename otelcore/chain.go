package otelcore

import (
	"context"
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	conntypes "github.com/cosmos/ibc-go/v8/modules/core/03-connection/types"
	chantypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"
	"github.com/hyperledger-labs/yui-relayer/core"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type Chain struct {
	core.Chain
	tracer trace.Tracer
}

func NewChain(chain core.Chain, tracer trace.Tracer) core.Chain {
	return &Chain{
		Chain:  chain,
		tracer: tracer,
	}
}

func UnwrapChain(chain core.Chain) (core.Chain, error) {
	c, ok := chain.(*Chain)
	if !ok {
		return nil, fmt.Errorf("chain type is not %T, but %T", &Chain{}, chain)
	}
	return c.Chain, nil
}

func (c *Chain) SendMsgs(ctx context.Context, msgs []sdk.Msg) ([]core.MsgID, error) {
	ctx, span := c.tracer.Start(ctx, "Chain.SendMsgs",
		core.WithChainAttributes(c.ChainID()),
	)
	defer span.End()

	ids, err := c.Chain.SendMsgs(ctx, msgs)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}
	return ids, err
}

func (c *Chain) GetMsgResult(ctx context.Context, id core.MsgID) (core.MsgResult, error) {
	ctx, span := c.tracer.Start(ctx, "Chain.GetMsgResult",
		core.WithChainAttributes(c.ChainID()),
	)
	defer span.End()

	result, err := c.Chain.GetMsgResult(ctx, id)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}
	return result, err
}

func (c *Chain) LatestHeight(ctx context.Context) (ibcexported.Height, error) {
	ctx, span := c.tracer.Start(ctx, "Chain.LatestHeight",
		core.WithChainAttributes(c.ChainID()),
	)
	defer span.End()

	height, err := c.Chain.LatestHeight(ctx)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}
	return height, err
}

func (c *Chain) Timestamp(ctx context.Context, height ibcexported.Height) (time.Time, error) {
	ctx, span := c.tracer.Start(ctx, "Chain.Timestamp",
		core.WithChainAttributes(c.ChainID()),
	)
	defer span.End()

	t, err := c.Chain.Timestamp(ctx, height)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}
	return t, err
}

func (c *Chain) QueryClientConsensusState(ctx core.QueryContext, dstClientConsHeight ibcexported.Height) (*clienttypes.QueryConsensusStateResponse, error) {
	ctx, span := core.StartTraceWithQueryContext(c.tracer, ctx, "Chain.QueryClientConsensusState",
		core.WithClientAttributes(c),
	)
	defer span.End()

	resp, err := c.Chain.QueryClientConsensusState(ctx, dstClientConsHeight)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}
	return resp, err
}

func (c *Chain) QueryClientState(ctx core.QueryContext) (*clienttypes.QueryClientStateResponse, error) {
	ctx, span := core.StartTraceWithQueryContext(c.tracer, ctx, "Chain.QueryClientState",
		core.WithChainAttributes(c.ChainID()),
	)
	defer span.End()

	resp, err := c.Chain.QueryClientState(ctx)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}
	return resp, err
}

func (c *Chain) QueryConnection(ctx core.QueryContext, connectionID string) (*conntypes.QueryConnectionResponse, error) {
	ctx, span := core.StartTraceWithQueryContext(c.tracer, ctx, "Chain.QueryConnection",
		core.WithClientAttributes(c),
		trace.WithAttributes(core.AttributeKeyConnectionID.String(connectionID)),
	)
	defer span.End()

	resp, err := c.Chain.QueryConnection(ctx, connectionID)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}
	return resp, err
}

func (c *Chain) QueryChannel(ctx core.QueryContext) (*chantypes.QueryChannelResponse, error) {
	ctx, span := core.StartTraceWithQueryContext(c.tracer, ctx, "Chain.QueryChannel",
		core.WithChannelAttributes(c),
	)
	defer span.End()

	resp, err := c.Chain.QueryChannel(ctx)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}
	return resp, err
}

func (c *Chain) QueryUnreceivedPackets(ctx core.QueryContext, seqs []uint64) ([]uint64, error) {
	ctx, span := core.StartTraceWithQueryContext(c.tracer, ctx, "Chain.QueryUnreceivedPackets",
		core.WithChannelAttributes(c),
	)
	defer span.End()

	packets, err := c.Chain.QueryUnreceivedPackets(ctx, seqs)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}
	return packets, err
}

func (c *Chain) QueryUnfinalizedRelayPackets(ctx core.QueryContext, counterparty core.LightClientICS04Querier) (core.PacketInfoList, error) {
	ctx, span := core.StartTraceWithQueryContext(c.tracer, ctx, "Chain.QueryUnfinalizedRelayPackets",
		core.WithChannelAttributes(c),
	)
	defer span.End()

	list, err := c.Chain.QueryUnfinalizedRelayPackets(ctx, counterparty)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}
	return list, err
}

func (c *Chain) QueryUnreceivedAcknowledgements(ctx core.QueryContext, seqs []uint64) ([]uint64, error) {
	ctx, span := core.StartTraceWithQueryContext(c.tracer, ctx, "Chain.QueryUnreceivedAcknowledgements",
		core.WithChannelAttributes(c),
	)
	defer span.End()

	acks, err := c.Chain.QueryUnreceivedAcknowledgements(ctx, seqs)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}
	return acks, err
}

func (c *Chain) QueryUnfinalizedRelayAcknowledgements(ctx core.QueryContext, counterparty core.LightClientICS04Querier) (core.PacketInfoList, error) {
	ctx, span := core.StartTraceWithQueryContext(c.tracer, ctx, "Chain.QueryUnfinalizedRelayAcknowledgements",
		core.WithChannelAttributes(c),
	)
	defer span.End()

	list, err := c.Chain.QueryUnfinalizedRelayAcknowledgements(ctx, counterparty)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}
	return list, err
}

func (c *Chain) QueryChannelUpgrade(ctx core.QueryContext) (*chantypes.QueryUpgradeResponse, error) {
	ctx, span := core.StartTraceWithQueryContext(c.tracer, ctx, "Chain.QueryChannelUpgrade",
		core.WithChannelAttributes(c),
	)
	defer span.End()

	resp, err := c.Chain.QueryChannelUpgrade(ctx)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}
	return resp, err
}

func (c *Chain) QueryChannelUpgradeError(ctx core.QueryContext) (*chantypes.QueryUpgradeErrorResponse, error) {
	ctx, span := core.StartTraceWithQueryContext(c.tracer, ctx, "Chain.QueryChannelUpgradeError",
		core.WithChannelAttributes(c),
	)
	defer span.End()

	resp, err := c.Chain.QueryChannelUpgradeError(ctx)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}
	return resp, err
}

func (c *Chain) QueryCanTransitionToFlushComplete(ctx core.QueryContext) (bool, error) {
	ctx, span := core.StartTraceWithQueryContext(c.tracer, ctx, "Chain.QueryCanTransitionToFlushComplete",
		core.WithChannelAttributes(c),
	)
	defer span.End()

	ok, err := c.Chain.QueryCanTransitionToFlushComplete(ctx)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}
	return ok, err
}

func (c *Chain) QueryBalance(ctx core.QueryContext, address sdk.AccAddress) (sdk.Coins, error) {
	ctx, span := core.StartTraceWithQueryContext(c.tracer, ctx, "Chain.QueryBalance",
		core.WithChainAttributes(c.ChainID()),
	)
	defer span.End()

	coins, err := c.Chain.QueryBalance(ctx, address)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}
	return coins, err
}

func (c *Chain) QueryDenomTraces(ctx core.QueryContext, offset, limit uint64) (*transfertypes.QueryDenomTracesResponse, error) {
	ctx, span := core.StartTraceWithQueryContext(c.tracer, ctx, "Chain.QueryDenomTraces",
		core.WithChainAttributes(c.ChainID()),
	)
	defer span.End()

	resp, err := c.Chain.QueryDenomTraces(ctx, offset, limit)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}
	return resp, err
}
