package core

import (
	"context"
	"fmt"

	"github.com/cosmos/ibc-go/v8/modules/core/exported"
	"github.com/hyperledger-labs/yui-relayer/internal/telemetry"
	"github.com/hyperledger-labs/yui-relayer/otelcore/semconv"
	"go.opentelemetry.io/otel/codes"
)

type Header interface {
	exported.ClientMessage
	GetHeight() exported.Height
}

// SyncHeaders manages the latest finalized headers on both `src` and `dst` chains
// It also provides the helper functions to update the clients on the chains
type SyncHeaders interface {
	// Updates updates the headers on both chains
	Updates(ctx context.Context, src, dst ChainInfoLightClient) error

	// GetLatestFinalizedHeader returns the latest finalized header of the chain
	GetLatestFinalizedHeader(chainID string) Header

	// GetQueryContext builds a query context based on the latest finalized header
	GetQueryContext(ctx context.Context, chainID string) QueryContext

	// SetupHeadersForUpdate returns `src` chain's headers needed to update the client on `dst` chain
	SetupHeadersForUpdate(ctx context.Context, src, dst ChainLightClient) ([]Header, error)

	// SetupBothHeadersForUpdate returns both `src` and `dst` chain's headers needed to update the clients on each chain
	SetupBothHeadersForUpdate(ctx context.Context, src, dst ChainLightClient) (srcHeaders, dstHeaders []Header, err error)
}

// ChainInfoLightClient = ChainInfo + LightClient
type ChainInfoLightClient interface {
	ChainInfo
	LightClient
}

// ChainLightClient = Chain + LightClient
type ChainLightClient interface {
	Chain
	LightClient
}

type syncHeaders struct {
	latestFinalizedHeaders map[string]Header // chainID => Header
}

var _ SyncHeaders = (*syncHeaders)(nil)

// NewSyncHeaders returns a new instance of SyncHeaders that can be easily
// kept "reasonably up to date"
func NewSyncHeaders(ctx context.Context, src, dst ChainInfoLightClient) (SyncHeaders, error) {
	logger := GetChainPairLogger(src, dst)
	if err := ensureDifferentChains(src, dst); err != nil {
		logger.ErrorContext(ctx, "error ensuring different chains", err)
		return nil, err
	}
	sh := &syncHeaders{
		latestFinalizedHeaders: map[string]Header{src.ChainID(): nil, dst.ChainID(): nil},
	}
	if err := sh.Updates(ctx, src, dst); err != nil {
		logger.ErrorContext(ctx, "error updating headers", err)
		return nil, err
	}
	return sh, nil
}

// Updates updates the headers on both chains
func (sh *syncHeaders) Updates(ctx context.Context, src, dst ChainInfoLightClient) error {
	ctx, span := tracer.Start(ctx, "syncHeaders.Updates", WithChainPairAttributes(src, dst))
	defer span.End()
	logger := GetChainPairLogger(src, dst)
	if err := ensureDifferentChains(src, dst); err != nil {
		logger.ErrorContext(ctx, "error ensuring different chains", err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	srcHeader, err := src.GetLatestFinalizedHeader(ctx)
	if err != nil {
		logger.ErrorContext(ctx, "error getting latest finalized header of src", err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	dstHeader, err := dst.GetLatestFinalizedHeader(ctx)
	if err != nil {
		logger.ErrorContext(ctx, "error getting latest finalized header of dst", err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	if err := sh.updateBlockMetrics(src, dst, srcHeader, dstHeader); err != nil {
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	sh.latestFinalizedHeaders[src.ChainID()] = srcHeader
	sh.latestFinalizedHeaders[dst.ChainID()] = dstHeader
	return nil
}

func (sh syncHeaders) updateBlockMetrics(src, dst ChainInfo, srcHeader, dstHeader Header) error {
	telemetry.ProcessedBlockHeightGauge.Set(
		int64(srcHeader.GetHeight().GetRevisionHeight()),
		semconv.ChainIDKey.String(src.ChainID()),
		semconv.DirectionKey.String("src"),
	)
	telemetry.ProcessedBlockHeightGauge.Set(
		int64(dstHeader.GetHeight().GetRevisionHeight()),
		semconv.ChainIDKey.String(dst.ChainID()),
		semconv.DirectionKey.String("dst"),
	)
	return nil
}

// GetLatestFinalizedHeader returns the latest finalized header of the chain
func (sh syncHeaders) GetLatestFinalizedHeader(chainID string) Header {
	return sh.latestFinalizedHeaders[chainID]
}

// GetQueryContext builds a query context based on the latest finalized header
func (sh syncHeaders) GetQueryContext(ctx context.Context, chainID string) QueryContext {
	return NewQueryContext(ctx, sh.GetLatestFinalizedHeader(chainID).GetHeight())
}

// SetupHeadersForUpdate returns `src` chain's headers to update the client on `dst` chain
func (sh syncHeaders) SetupHeadersForUpdate(ctx context.Context, src, dst ChainLightClient) ([]Header, error) {
	logger := GetChainPairLogger(src, dst)
	if err := ensureDifferentChains(src, dst); err != nil {
		logger.ErrorContext(ctx, "error ensuring different chains", err)
		return nil, err
	}
	return SetupHeadersForUpdateSync(src, ctx, dst, sh.GetLatestFinalizedHeader(src.ChainID()))
}

// SetupBothHeadersForUpdate returns both `src` and `dst` chain's headers to update the clients on each chain
func (sh syncHeaders) SetupBothHeadersForUpdate(ctx context.Context, src, dst ChainLightClient) ([]Header, []Header, error) {
	logger := GetChainPairLogger(src, dst)
	srcHs, err := sh.SetupHeadersForUpdate(ctx, src, dst)
	if err != nil {
		logger.ErrorContext(ctx, "error setting up headers for update on src", err)
		return nil, nil, err
	}
	dstHs, err := sh.SetupHeadersForUpdate(ctx, dst, src)
	if err != nil {
		logger.ErrorContext(ctx, "error setting up headers for update on dst", err)
		return nil, nil, err
	}
	return srcHs, dstHs, nil
}

func ensureDifferentChains(src, dst ChainInfo) error {
	if src.ChainID() == dst.ChainID() {
		return fmt.Errorf("the two chains are probably the same.: src=%v dst=%v", src.ChainID(), dst.ChainID())
	} else {
		return nil
	}
}
