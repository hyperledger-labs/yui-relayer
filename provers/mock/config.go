package mock

import (
	"time"

	"github.com/hyperledger-labs/yui-relayer/core"
)

var _ core.ProverConfigI = (*ProverConfig)(nil)

func (c *ProverConfig) Build(chain core.ChainI) (core.ProverI, error) {
	return NewProver(chain, uint64(time.Now().UnixNano())), nil
}
