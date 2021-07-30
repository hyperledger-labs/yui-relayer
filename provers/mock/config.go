package mock

import (
	"github.com/hyperledger-labs/yui-relayer/core"
)

var _ core.ProverConfigI = (*ProverConfig)(nil)

func (c *ProverConfig) Build(chain core.ChainI) (core.ProverI, error) {
	return NewProver(chain, 1), nil
}
