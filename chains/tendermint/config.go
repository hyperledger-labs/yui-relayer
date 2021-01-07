package tendermint

import (
	"github.com/cosmos/relayer/relayer"
	"github.com/datachainlab/relayer/core"
)

var _ core.ChainConfigI = (*ChainConfig)(nil)

func (c ChainConfig) GetChain() core.ChainI {
	return &Chain{
		base: relayer.Chain{
			Key:            c.Key,
			ChainID:        c.ChainId,
			RPCAddr:        c.RpcAddr,
			AccountPrefix:  c.AccountPrefix,
			GasAdjustment:  float64(c.GasAdjustment),
			GasPrices:      c.GasPrices,
			TrustingPeriod: c.TrustingPeriod,
		},
	}
}
