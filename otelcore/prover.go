package otelcore

import (
	"context"
	"fmt"

	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"
	"github.com/hyperledger-labs/yui-relayer/core"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type Prover struct {
	core.Prover
	chainID string
	tracer  trace.Tracer
}

func NewProver(prover core.Prover, chainID string, tracer trace.Tracer) core.Prover {
	return &Prover{
		Prover:  prover,
		chainID: chainID,
		tracer:  tracer,
	}
}

func UnwrapProver(prover core.Prover) (core.Prover, error) {
	p, ok := prover.(*Prover)
	if !ok {
		return nil, fmt.Errorf("prover type is not %T, but %T", &Prover{}, prover)
	}
	return p.Prover, nil
}

func (p *Prover) GetLatestFinalizedHeader(ctx context.Context) (core.Header, error) {
	ctx, span := p.tracer.Start(ctx, "Prover.GetLatestFinalizedHeader",
		core.WithChainAttributes(p.chainID),
	)
	defer span.End()

	header, err := p.Prover.GetLatestFinalizedHeader(ctx)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}
	return header, err
}

func (p *Prover) CreateInitialLightClientState(ctx context.Context, height ibcexported.Height) (ibcexported.ClientState, ibcexported.ConsensusState, error) {
	ctx, span := p.tracer.Start(ctx, "Prover.CreateInitialLightClientState",
		core.WithChainAttributes(p.chainID),
	)
	defer span.End()

	clientState, consensusState, err := p.Prover.CreateInitialLightClientState(ctx, height)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}
	return clientState, consensusState, err
}

func (p *Prover) SetupHeadersForUpdate(ctx context.Context, counterparty core.FinalityAwareChain, latestFinalizedHeader core.Header) ([]core.Header, error) {
	ctx, span := p.tracer.Start(ctx, "Prover.SetupHeadersForUpdate",
		core.WithChainAttributes(p.chainID),
		trace.WithAttributes(attribute.String("counterparty_chain_id", counterparty.ChainID())),
	)
	defer span.End()

	headers, err := p.Prover.SetupHeadersForUpdate(ctx, counterparty, latestFinalizedHeader)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}
	return headers, err
}

func (p *Prover) CheckRefreshRequired(ctx context.Context, counterparty core.ChainInfoICS02Querier) (bool, error) {
	ctx, span := p.tracer.Start(ctx, "Prover.CheckRefreshRequired",
		core.WithChainAttributes(p.chainID),
		trace.WithAttributes(attribute.String("counterparty_chain_id", counterparty.ChainID())),
	)
	defer span.End()

	required, err := p.Prover.CheckRefreshRequired(ctx, counterparty)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}
	return required, err
}

func (p *Prover) ProveState(ctx core.QueryContext, path string, value []byte) ([]byte, clienttypes.Height, error) {
	ctx, span := core.StartTraceWithQueryContext(p.tracer, ctx, "Prover.ProveState",
		core.WithChainAttributes(p.chainID),
		trace.WithAttributes(core.AttributeKeyPath.String(path)),
	)
	defer span.End()

	proof, height, err := p.Prover.ProveState(ctx, path, value)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}
	return proof, height, err
}

func (p *Prover) ProveHostConsensusState(ctx core.QueryContext, height ibcexported.Height, consensusState ibcexported.ConsensusState) ([]byte, error) {
	ctx, span := core.StartTraceWithQueryContext(p.tracer, ctx, "Prover.ProveHostConsensusState",
		core.WithChainAttributes(p.chainID),
	)
	defer span.End()

	proof, err := p.Prover.ProveHostConsensusState(ctx, height, consensusState)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}
	return proof, err
}
