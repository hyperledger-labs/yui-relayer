package fabric

import (
	"github.com/datachainlab/relayer/core"
)

var _ core.ChainConfigI = (*ChainConfig)(nil)

func (c ChainConfig) GetChain() core.ChainI {
	return NewChain(c)
}
