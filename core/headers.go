package core

import "github.com/cosmos/cosmos-sdk/x/ibc/core/exported"

type SyncHeadersI interface {
	GetHeight(chainID string) uint64
	GetHeader(chainID string) HeaderI
}

type HeaderI interface {
	GetHeight() exported.Height
	// Update() error
}

// NewSyncHeaders returns a new instance of map[string]*tmclient.Header that can be easily
// kept "reasonably up to date"
func NewSyncHeaders(src, dst ChainI) (SyncHeadersI, error) {
	srch, dsth, err := UpdatesWithHeaders(src, dst)
	if err != nil {
		return nil, err
	}
	// return &SyncHeaders{hds: map[string]*tmclient.Header{src.ChainID: srch, dst.ChainID: dsth}}, nil
	_, _ = srch, dsth
	panic("not implemented error")
}
