package fabric

import (
	"context"

	"github.com/hyperledger-labs/yui-relayer/core"
)

func (srcChain *Chain) CreateTrustedHeader(_ context.Context, dstChain core.ChainI, srcHeader core.HeaderI) (core.HeaderI, error) {
	return nil, nil
}

func (srcChain *Chain) UpdateLightWithHeader(ctx context.Context) (core.HeaderI, error) {
	return srcChain.QueryLatestHeader(ctx)
}
