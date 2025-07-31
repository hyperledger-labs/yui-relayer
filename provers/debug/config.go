package debug

import (
	"fmt"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/hyperledger-labs/yui-relayer/core"
)

var _ core.ProverConfig = (*ProverConfig)(nil)

var _ codectypes.UnpackInterfacesMessage = (*ProverConfig)(nil)

func (cfg *ProverConfig) UnpackInterfaces(unpacker codectypes.AnyUnpacker) error {
	if cfg == nil {
		return nil
	}
	if err := unpacker.UnpackAny(cfg.OriginProver, new(core.ProverConfig)); err != nil {
		return err
	}
	return nil
}

func (pc ProverConfig) Build(chain core.Chain) (core.Prover, error) {
	if err := pc.Validate(); err != nil {
		return nil, err
	}

	if pc.OriginProver == nil {
		return nil, fmt.Errorf("OriginProver must set")
	}
	if pc.OriginProver.GetCachedValue() == nil {
		return nil, fmt.Errorf("OriginProver.GetCachedValue() must set")
	}
	originProver, err := pc.OriginProver.GetCachedValue().(core.ProverConfig).Build(chain)
	if err != nil {
		return nil, err
	}
	prover := NewProver(chain, pc, originProver)
	return prover, nil
}

func (c ProverConfig) Validate() error {
	return nil
}
