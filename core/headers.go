package core

import (
	"context"

	"github.com/cosmos/ibc-go/v4/modules/core/exported"
)

type HeaderI interface {
	exported.Header
}

type SyncHeadersI interface {
	GetLatestFinalizedHeader(chainID string) HeaderI

	GetQueryContext(chainID string) QueryContext
	GetQueryProofContext(chainID string) QueryProofContext

	// SetupHeadersForUpdate returns the latest header of light client
	SetupHeadersForUpdate(src, dst *ProvableChain) ([]HeaderI, error)
	// SetupBothHeadersForUpdate returns the latest headers for both src and dst client.
	SetupBothHeadersForUpdate(src, dst *ProvableChain) (srcHeaders []HeaderI, dstHeaders []HeaderI, err error)

	// Updates updates the header of light client
	Updates(src LightClientI, dst LightClientI) error
}

type syncHeaders struct {
	latestFinalizedHeaders map[string]HeaderI // chainID => HeaderI
}

var _ SyncHeadersI = (*syncHeaders)(nil)

// NewSyncHeaders returns a new instance of SyncHeadersI that can be easily
// kept "reasonably up to date"
func NewSyncHeaders(src, dst LightClientI) (SyncHeadersI, error) {
	sh := &syncHeaders{
		latestFinalizedHeaders: map[string]HeaderI{src.GetChainID(): nil, dst.GetChainID(): nil},
	}
	if err := sh.Updates(src, dst); err != nil {
		return nil, err
	}
	return sh, nil
}

func (sh syncHeaders) GetLatestFinalizedHeader(chainID string) HeaderI {
	return sh.latestFinalizedHeaders[chainID]
}

func (sh syncHeaders) GetQueryContext(chainID string) QueryContext {
	return NewQueryContext(context.TODO(), sh.latestFinalizedHeaders[chainID].GetHeight())
}

func (sh syncHeaders) GetQueryProofContext(chainID string) QueryProofContext {
	return NewQueryProofContext(context.TODO(), sh.latestFinalizedHeaders[chainID])
}

// SetupHeadersForUpdate implements SyncHeadersI
func (sh syncHeaders) SetupHeadersForUpdate(src, dst *ProvableChain) ([]HeaderI, error) {
	return src.SetupHeadersForUpdate(dst, sh.latestFinalizedHeaders[src.GetChainID()])
}

// SetupBothHeadersForUpdate implements SyncHeadersI
func (sh syncHeaders) SetupBothHeadersForUpdate(src, dst *ProvableChain) ([]HeaderI, []HeaderI, error) {
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

// Updates implements SyncHeadersI
func (sh *syncHeaders) Updates(src, dst LightClientI) error {
	srcHeader, err := src.GetLatestFinalizedHeader()
	if err != nil {
		return err
	}
	dstHeader, err := dst.GetLatestFinalizedHeader()
	if err != nil {
		return err
	}

	sh.latestFinalizedHeaders[src.GetChainID()] = srcHeader
	sh.latestFinalizedHeaders[dst.GetChainID()] = dstHeader
	return nil
}
