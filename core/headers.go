package core

import (
	"context"
	"fmt"

	"github.com/cosmos/ibc-go/v7/modules/core/exported"
	"github.com/hyperledger-labs/yui-relayer/log"
)

type Header interface {
	exported.ClientMessage
	GetHeight() exported.Height
}

// SyncHeaders manages the latest finalized headers on both `src` and `dst` chains
// It also provides the helper functions to update the clients on the chains
type SyncHeaders interface {
	// Updates updates the headers on both chains
	Updates(src, dst ChainInfoLightClient) error

	// GetLatestFinalizedHeader returns the latest finalized header of the chain
	GetLatestFinalizedHeader(chainID string) Header

	// GetQueryContext builds a query context based on the latest finalized header
	GetQueryContext(chainID string) QueryContext

	// SetupHeadersForUpdate returns `src` chain's headers needed to update the client on `dst` chain
	SetupHeadersForUpdate(src, dst ChainICS02QuerierLightClient) ([]Header, error)

	// SetupBothHeadersForUpdate returns both `src` and `dst` chain's headers needed to update the clients on each chain
	SetupBothHeadersForUpdate(src, dst ChainICS02QuerierLightClient) (srcHeaders []Header, dstHeaders []Header, err error)
}

// ChainInfoLightClient = ChainInfo + LightClient
type ChainInfoLightClient interface {
	ChainInfo
	LightClient
}

// ChainICS02QuerierLightClient = ChainInfoLightClient + ICS02Querier
type ChainICS02QuerierLightClient interface {
	ChainInfoLightClient
	ICS02Querier
}

type syncHeaders struct {
	latestFinalizedHeaders map[string]Header // chainID => Header
}

var _ SyncHeaders = (*syncHeaders)(nil)

// NewSyncHeaders returns a new instance of SyncHeaders that can be easily
// kept "reasonably up to date"
func NewSyncHeaders(src, dst ChainInfoLightClient) (SyncHeaders, error) {
	relayLogger := log.GetLogger()
	if err := ensureDifferentChains(src, dst); err != nil {
		relayLogger.Error("error ensuring different chains", err)
		return nil, err
	}
	sh := &syncHeaders{
		latestFinalizedHeaders: map[string]Header{src.ChainID(): nil, dst.ChainID(): nil},
	}
	if err := sh.Updates(src, dst); err != nil {
		relayLogger.Error("error updating headers", err)
		return nil, err
	}
	return sh, nil
}

// Updates updates the headers on both chains
func (sh *syncHeaders) Updates(src, dst ChainInfoLightClient) error {
	relayLogger := log.GetLogger()
	if err := ensureDifferentChains(src, dst); err != nil {
		relayLogger.Error("error ensuring different chains", err)
		return err
	}

	srcHeader, err := src.GetLatestFinalizedHeader()
	if err != nil {
		relayLogger.Error("error getting latest finalized header of src", err)
		return err
	}
	dstHeader, err := dst.GetLatestFinalizedHeader()
	if err != nil {
		relayLogger.Error("error getting latest finalized header of dst", err)
		return err
	}

	sh.latestFinalizedHeaders[src.ChainID()] = srcHeader
	sh.latestFinalizedHeaders[dst.ChainID()] = dstHeader
	return nil
}

// GetLatestFinalizedHeader returns the latest finalized header of the chain
func (sh syncHeaders) GetLatestFinalizedHeader(chainID string) Header {
	return sh.latestFinalizedHeaders[chainID]
}

// GetQueryContext builds a query context based on the latest finalized header
func (sh syncHeaders) GetQueryContext(chainID string) QueryContext {
	return NewQueryContext(context.TODO(), sh.GetLatestFinalizedHeader(chainID).GetHeight())
}

// SetupHeadersForUpdate returns `src` chain's headers to update the client on `dst` chain
func (sh syncHeaders) SetupHeadersForUpdate(src, dst ChainICS02QuerierLightClient) ([]Header, error) {
	relayLogger := log.GetLogger()
	if err := ensureDifferentChains(src, dst); err != nil {
		relayLogger.Error("error ensuring different chains", err)
		return nil, err
	}
	return src.SetupHeadersForUpdate(dst, sh.GetLatestFinalizedHeader(src.ChainID()))
}

// SetupBothHeadersForUpdate returns both `src` and `dst` chain's headers to update the clients on each chain
func (sh syncHeaders) SetupBothHeadersForUpdate(src, dst ChainICS02QuerierLightClient) ([]Header, []Header, error) {
	relayLogger := log.GetLogger()
	srcHs, err := sh.SetupHeadersForUpdate(src, dst)
	if err != nil {
		relayLogger.Error("error setting up headers for update on src", err)
		return nil, nil, err
	}
	dstHs, err := sh.SetupHeadersForUpdate(dst, src)
	if err != nil {
		relayLogger.Error("error setting up headers for update on dst", err)
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
