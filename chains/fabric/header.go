package fabric

import (
	"github.com/hyperledger-labs/yui-relayer/core"
)

func (srcChain *Chain) CreateTrustedHeader(dstChain core.ChainI, srcHeader core.HeaderI) (core.HeaderI, error) {
	return nil, nil
}

func (srcChain *Chain) UpdateLightWithHeader() (core.HeaderI, error) {
	return srcChain.QueryLatestHeader()
}
