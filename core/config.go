package core

import (
	"encoding/json"
	"errors"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/gogoproto/proto"
	"github.com/hyperledger-labs/yui-relayer/log"
	"github.com/hyperledger-labs/yui-relayer/utils"
)

// ChainProverConfig defines the top level configuration for a chain instance
type ChainProverConfig struct {
	Chain  json.RawMessage `json:"chain" yaml:"chain"` // NOTE: it's any type as json format
	Prover json.RawMessage `json:"prover" yaml:"prover"`

	// cache
	chain  ChainConfig  `json:"-" yaml:"-"`
	prover ProverConfig `json:"-" yaml:"-"`
}

// ChainConfig defines a chain configuration and its builder
type ChainConfig interface {
	proto.Message
	Build() (Chain, error)
}

// ProverConfig defines a prover configuration and its builder
type ProverConfig interface {
	proto.Message
	Build(Chain) (Prover, error)
}

// NewChainProverConfig returns a new config instance
func NewChainProverConfig(m codec.JSONCodec, chain ChainConfig, client ProverConfig) (*ChainProverConfig, error) {
	relayLogger := log.GetLogger()
	cbz, err := utils.MarshalJSONAny(m, chain)
	if err != nil {
		relayLogger.Error("error marshalling chain config", err)
		return nil, err
	}
	clbz, err := utils.MarshalJSONAny(m, client)
	if err != nil {
		relayLogger.Error("error marshalling client config", err)
		return nil, err
	}
	return &ChainProverConfig{
		Chain:  cbz,
		Prover: clbz,
		chain:  chain,
		prover: client,
	}, nil
}

// Init initialises the configuration
func (cc *ChainProverConfig) Init(m codec.Codec) error {
	var chain ChainConfig
	if err := utils.UnmarshalJSONAny(m, &chain, cc.Chain); err != nil {
		return err
	}
	var prover ProverConfig
	if err := utils.UnmarshalJSONAny(m, &prover, cc.Prover); err != nil {
		return err
	}
	cc.chain = chain
	cc.prover = prover
	return nil
}

// GetChainConfig returns the cached ChainConfig instance
func (cc ChainProverConfig) GetChainConfig() (ChainConfig, error) {
	if cc.chain == nil {
		return nil, errors.New("chain is nil")
	}
	return cc.chain, nil
}

// GetProverConfig returns the cached ChainProverConfig instance
func (cc ChainProverConfig) GetProverConfig() (ProverConfig, error) {
	if cc.prover == nil {
		return nil, errors.New("client is nil")
	}
	return cc.prover, nil
}

// Build returns a new ProvableChain instance
func (cc ChainProverConfig) Build() (*ProvableChain, error) {
	chainConfig, err := cc.GetChainConfig()
	if err != nil {
		return nil, err
	}
	proverConfig, err := cc.GetProverConfig()
	if err != nil {
		return nil, err
	}
	chain, err := chainConfig.Build()
	if err != nil {
		return nil, err
	}
	prover, err := proverConfig.Build(chain)
	if err != nil {
		return nil, err
	}
	return NewProvableChain(chain, prover), nil
}
