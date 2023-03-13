package core

import (
	"context"

	"github.com/cosmos/ibc-go/v4/modules/core/exported"
)

type HeaderI interface {
	exported.Header
}

// SyncHeadersI manages the latest finalized headers on both `src` and `dst` chains
// It also provides the helper functions to update the clients on the chains
type SyncHeadersI interface {
	// Updates updates the headers on both chains
	Updates(src, dst ChainQuerierLightClient) error

	// GetLatestFinalizedHeader returns the latest finalized header of the chain
	GetLatestFinalizedHeader(chainID string) HeaderI

	// GetQueryContext builds a query context based on the latest finalized header
	GetQueryContext(chainID string) QueryContext

	// SetupHeadersForUpdate returns `src` chain's headers needed to update the client on `dst` chain
	SetupHeadersForUpdate(src, dst ChainICS02QuerierLightClient) ([]HeaderI, error)

	// SetupBothHeadersForUpdate returns both `src` and `dst` chain's headers needed to update the clients on each chain
	SetupBothHeadersForUpdate(src, dst ChainICS02QuerierLightClient) (srcHeaders []HeaderI, dstHeaders []HeaderI, err error)
}

// ChainQuerierLightClient = ChainQuerier + LightClient
type ChainQuerierLightClient interface {
	ChainQuerier
	LightClient
}

// ChainICS02QuerierLightClient = ChainQuerierLightClient + ICS02Querier
type ChainICS02QuerierLightClient interface {
	ChainQuerierLightClient
	ICS02Querier
}

type syncHeaders struct {
	latestFinalizedHeaders map[string]HeaderI // chainID => HeaderI
}

var _ SyncHeadersI = (*syncHeaders)(nil)

// NewSyncHeaders returns a new instance of SyncHeadersI that can be easily
// kept "reasonably up to date"
func NewSyncHeaders(src, dst ChainQuerierLightClient) (SyncHeadersI, error) {
	sh := &syncHeaders{
		latestFinalizedHeaders: map[string]HeaderI{src.ChainID(): nil, dst.ChainID(): nil},
	}
	if err := sh.Updates(src, dst); err != nil {
		return nil, err
	}
	return sh, nil
}

// Updates updates the headers on both chains
func (sh *syncHeaders) Updates(src, dst ChainQuerierLightClient) error {
	srcHeader, err := src.GetLatestFinalizedHeader()
	if err != nil {
		return err
	}
	dstHeader, err := dst.GetLatestFinalizedHeader()
	if err != nil {
		return err
	}

	sh.latestFinalizedHeaders[src.ChainID()] = srcHeader
	sh.latestFinalizedHeaders[dst.ChainID()] = dstHeader
	return nil
}

// GetLatestFinalizedHeader returns the latest finalized header of the chain
func (sh syncHeaders) GetLatestFinalizedHeader(chainID string) HeaderI {
	return sh.latestFinalizedHeaders[chainID]
}

// GetQueryContext builds a query context based on the latest finalized header
func (sh syncHeaders) GetQueryContext(chainID string) QueryContext {
	return NewQueryContext(context.TODO(), sh.GetLatestFinalizedHeader(chainID).GetHeight())
}

// SetupHeadersForUpdate returns `src` chain's headers to update the client on `dst` chain
func (sh syncHeaders) SetupHeadersForUpdate(src, dst ChainICS02QuerierLightClient) ([]HeaderI, error) {
	return src.SetupHeadersForUpdate(dst, sh.GetLatestFinalizedHeader(src.ChainID()))
}

// SetupBothHeadersForUpdate returns both `src` and `dst` chain's headers to update the clients on each chain
func (sh syncHeaders) SetupBothHeadersForUpdate(src, dst ChainICS02QuerierLightClient) ([]HeaderI, []HeaderI, error) {
	srcHs, err := sh.SetupHeadersForUpdate(src, dst)
	if err != nil {
		return nil, nil, err
	}
	dstHs, err := sh.SetupHeadersForUpdate(dst, src)
	if err != nil {
		return nil, nil, err
	}
	return srcHs, dstHs, nil
}
