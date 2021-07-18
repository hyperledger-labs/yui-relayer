package ethereum

import "github.com/hyperledger-labs/yui-relayer/core"

var _ core.ChainConfigI = (*ChainConfig)(nil)

func (c ChainConfig) Build() (core.ChainI, error) {
	return &Chain{
		config: c,
	}, nil
}
