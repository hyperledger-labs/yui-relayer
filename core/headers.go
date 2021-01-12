package core

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/x/ibc/core/exported"
)

type SyncHeadersI interface {
	GetHeight(chainID string) uint64
	GetHeader(chainID string) HeaderI
	GetTrustedHeaders(src, dst ChainI) (srcHeader HeaderI, dstHeader HeaderI, err error)
	Updates(ChainI, ChainI) error
}

var _ SyncHeadersI = (*syncHeaders)(nil)

func (sh syncHeaders) GetHeight(chainID string) uint64 {
	return sh.hds[chainID].GetHeight().GetVersionHeight()
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
		fmt.Println("failed to GetTrustedHeaders(src):", err)
		return nil, nil, err
	}
	return srcTh, dstTh, nil
}

func (sh *syncHeaders) Updates(src, dst ChainI) error {
	// TODO should we introduce light client DB?
	// original implementation uses it to manage the latest header

	srch, dsth, err := UpdatesWithHeaders(src, dst)
	if err != nil {
		return err
	}
	sh.hds = map[string]HeaderI{src.ChainID(): srch, dst.ChainID(): dsth}
	sh.chains = map[string]ChainI{src.ChainID(): src, dst.ChainID(): dst}
	return nil
}

type syncHeaders struct {
	hds    map[string]HeaderI
	chains map[string]ChainI
}

type HeaderI interface {
	exported.Header
}

// NewSyncHeaders returns a new instance of SyncHeadersI that can be easily
// kept "reasonably up to date"
func NewSyncHeaders(src, dst ChainI) (SyncHeadersI, error) {
	srch, dsth, err := UpdatesWithHeaders(src, dst)
	if err != nil {
		return nil, err
	}
	return &syncHeaders{
		hds:    map[string]HeaderI{src.ChainID(): srch, dst.ChainID(): dsth},
		chains: map[string]ChainI{src.ChainID(): src, dst.ChainID(): dst},
	}, nil
}
