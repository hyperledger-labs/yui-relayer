package core

import (
	"github.com/cosmos/ibc-go/modules/core/exported"
)

type HeaderI interface {
	exported.Header
}

type SyncHeadersI interface {
	// GetProvableHeight returns the provable height of chain
	GetProvableHeight(chainID string) int64
	// GetQueryableHeight returns the queryable height of chain
	GetQueryableHeight(chainID string) int64
	// GetHeader returns the latest header of light client
	GetHeader(src, dst LightClientIBCQueryierI) (HeaderI, error)
	// GetHeaders returns the latest headers for both src and dst client.
	GetHeaders(src, dst LightClientIBCQueryierI) (srcHeader HeaderI, dstHeader HeaderI, err error)
	// Updates updates the header of light client
	Updates(src LightClientI, dst LightClientI) error
}

type syncHeaders struct {
	latestHeaders          map[string]HeaderI // chainID => HeaderI
	latestProvableHeights  map[string]int64   // chainID => height
	latestQueryableHeights map[string]int64   // chainID => height
}

var _ SyncHeadersI = (*syncHeaders)(nil)

// NewSyncHeaders returns a new instance of SyncHeadersI that can be easily
// kept "reasonably up to date"
func NewSyncHeaders(src, dst LightClientI) (SyncHeadersI, error) {
	sh := &syncHeaders{
		latestHeaders:          map[string]HeaderI{src.GetChainID(): nil, dst.GetChainID(): nil},
		latestProvableHeights:  map[string]int64{src.GetChainID(): 0, dst.GetChainID(): 0},
		latestQueryableHeights: map[string]int64{src.GetChainID(): 0, dst.GetChainID(): 0},
	}
	if err := sh.Updates(src, dst); err != nil {
		return nil, err
	}
	return sh, nil
}

// GetProvableHeight implements SyncHeadersI
func (sh syncHeaders) GetProvableHeight(chainID string) int64 {
	return sh.latestProvableHeights[chainID]
}

// GetQueryableHeight implements SyncHeadersI
func (sh syncHeaders) GetQueryableHeight(chainID string) int64 {
	return sh.latestQueryableHeights[chainID]
}

// GetHeader implements SyncHeadersI
func (sh syncHeaders) GetHeader(src, dst LightClientIBCQueryierI) (HeaderI, error) {
	return src.SetupHeader(dst, sh.latestHeaders[src.GetChainID()])
}

// GetHeaders implements SyncHeadersI
func (sh syncHeaders) GetHeaders(src, dst LightClientIBCQueryierI) (HeaderI, HeaderI, error) {
	srcTh, err := sh.GetHeader(src, dst)
	if err != nil {
		return nil, nil, err
	}
	dstTh, err := sh.GetHeader(dst, src)
	if err != nil {
		return nil, nil, err
	}
	return srcTh, dstTh, nil
}

// Updates implements SyncHeadersI
func (sh *syncHeaders) Updates(src, dst LightClientI) error {
	srcHeader, srcPHeight, srcQHeight, err := src.UpdateLightWithHeader()
	if err != nil {
		return err
	}
	dstHeader, dstPHeight, dstQHeight, err := dst.UpdateLightWithHeader()
	if err != nil {
		return err
	}

	sh.latestHeaders[src.GetChainID()] = srcHeader
	sh.latestHeaders[dst.GetChainID()] = dstHeader

	sh.latestProvableHeights[src.GetChainID()] = srcPHeight
	sh.latestProvableHeights[dst.GetChainID()] = dstPHeight

	sh.latestQueryableHeights[src.GetChainID()] = srcQHeight
	sh.latestQueryableHeights[dst.GetChainID()] = dstQHeight
	return nil
}
