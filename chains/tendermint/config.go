package tendermint

import (
	"fmt"

	"github.com/hyperledger-labs/yui-relayer/core"
)

var _ core.ChainConfig = (*ChainConfig)(nil)

func (c ChainConfig) Build() (core.Chain, error) {
	return &Chain{
		config: c,
	}, nil
}

var _ core.ProverConfig = (*ProverConfig)(nil)

func (c ProverConfig) Build(chain core.Chain) (core.Prover, error) {
	chain_, ok := chain.(*Chain)
	if !ok {
		return nil, fmt.Errorf("chain type must be %T, not %T", &Chain{}, chain)
	}
	return NewProver(chain_, c), nil
}
