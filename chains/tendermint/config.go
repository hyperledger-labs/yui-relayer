package tendermint

import (
	"github.com/datachainlab/relayer/core"
)

var _ core.ChainConfigI = (*ChainConfig)(nil)

func (c ChainConfig) GetChain() core.ChainI {
	return &Chain{
		config: c,
	}
}
