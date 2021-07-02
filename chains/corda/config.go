package corda

import (
	"github.com/hyperledger-labs/yui-relayer/core"
)

var _ core.ChainConfigI = (*ChainConfig)(nil)

func (c ChainConfig) GetChain() core.ChainI {
	return NewChain(c)
}
