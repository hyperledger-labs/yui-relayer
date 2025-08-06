package debug

import (
	"fmt"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/hyperledger-labs/yui-relayer/core"
)

var _ core.ChainConfig = (*ChainConfig)(nil)

var _ codectypes.UnpackInterfacesMessage = (*ChainConfig)(nil)

func (cfg *ChainConfig) UnpackInterfaces(unpacker codectypes.AnyUnpacker) error {
	if cfg == nil {
		return nil
	}
	if err := unpacker.UnpackAny(cfg.OriginChain, new(core.ChainConfig)); err != nil {
		return err
	}
	return nil
}


func (c ChainConfig) Build() (core.Chain, error) {
	if c.OriginChain == nil {
		return nil, fmt.Errorf("OriginChain must set")
	}
	if c.OriginChain.GetCachedValue() == nil {
		return nil, fmt.Errorf("OriginChain.GetCachedValue() must be set")
	}

	originChain, err := c.OriginChain.GetCachedValue().(core.ChainConfig).Build()
	if err != nil {
		return nil, err
	}

	return &Chain {
		config: c,
		OriginChain: originChain,
	}, nil
}

func (c ChainConfig) Validate() error {
	return nil
}
