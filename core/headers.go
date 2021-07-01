package core

import (
	"fmt"

	"github.com/cosmos/ibc-go/modules/core/exported"
)

type HeaderI interface {
	exported.Header
}

type SyncHeadersI interface {
	GetHeight(chainID string) uint64
	GetHeader(chainID string) HeaderI
	GetTrustedHeaders(src, dst ChainI) (srcHeader HeaderI, dstHeader HeaderI, err error)
	Updates(ChainI, ChainI) error
}

type syncHeaders struct {
	hds map[string]HeaderI
}

var _ SyncHeadersI = (*syncHeaders)(nil)

// NewSyncHeaders returns a new instance of SyncHeadersI that can be easily
// kept "reasonably up to date"
func NewSyncHeaders(src, dst ChainI) (SyncHeadersI, error) {
	srch, dsth, err := UpdatesWithHeaders(src, dst)
	if err != nil {
		return nil, err
	}
	return &syncHeaders{
		hds: map[string]HeaderI{src.ChainID(): srch, dst.ChainID(): dsth},
	}, nil
}

func (sh syncHeaders) GetHeight(chainID string) uint64 {
	return sh.hds[chainID].GetHeight().GetRevisionHeight()
}

func (sh syncHeaders) GetHeader(chainID string) HeaderI {
	return sh.hds[chainID]
}

func (sh syncHeaders) GetTrustedHeaders(src, dst ChainI) (HeaderI, HeaderI, error) {
	srcTh, err := src.CreateTrustedHeader(dst, sh.GetHeader(src.ChainID()))
	if err != nil {
		fmt.Println("failed to GetTrustedHeaders(src):", err)
		return nil, nil, err
	}
	dstTh, err := dst.CreateTrustedHeader(src, sh.GetHeader(dst.ChainID()))
	if err != nil {
		fmt.Println("failed to GetTrustedHeaders(dst):", err)
		return nil, nil, err
	}
	return srcTh, dstTh, nil
}

func (sh *syncHeaders) Updates(src, dst ChainI) error {
	srch, err := src.UpdateLightWithHeader()
	if err != nil {
		return err
	}
	dsth, err := dst.UpdateLightWithHeader()
	if err != nil {
		return err
	}
	sh.hds = map[string]HeaderI{src.ChainID(): srch, dst.ChainID(): dsth}
	return nil
}
