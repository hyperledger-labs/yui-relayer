package tendermint

import (
	"fmt"

	"github.com/hyperledger-labs/yui-relayer/core"
)

var _ core.ChainConfigI = (*ChainConfig)(nil)

func (c ChainConfig) Build() (core.ChainI, error) {
	return &Chain{
		config: c,
	}, nil
}

var _ core.ProverConfigI = (*ProverConfig)(nil)

func (c ProverConfig) Build(chain core.ChainI) (core.ProverI, error) {
	chain_, ok := chain.(*Chain)
	if !ok {
		return nil, fmt.Errorf("chain type must be %T, not %T", &Chain{}, chain)
	}
	return NewProver(chain_, c), nil
}
