package core

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/gogoproto/proto"
	"github.com/hyperledger-labs/yui-relayer/log"
	"github.com/hyperledger-labs/yui-relayer/utils"
)

var config ConfigI

type ConfigIDType string

const (
	ConfigIDClient     ConfigIDType = "client"
	ConfigIDConnection ConfigIDType = "connection"
	ConfigIDChannel    ConfigIDType = "channel"
)

type ConfigI interface {
	UpdateConfigID(pathName string, chainID string, configID ConfigIDType, id string) error
}

func SetCoreConfig(c ConfigI) {
	if config != nil {
		panic("core config already set")
	}
	config = c
}

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
	Validate() error
}

// ProverConfig defines a prover configuration and its builder
type ProverConfig interface {
	proto.Message
	Build(Chain) (Prover, error)
	Validate() error
}

// NewChainProverConfig returns a new config instance
func NewChainProverConfig(m codec.JSONCodec, chain ChainConfig, client ProverConfig) (*ChainProverConfig, error) {
	logger := log.GetLogger().WithModule("core.config")
	cbz, err := utils.MarshalJSONAny(m, chain)
	if err != nil {
		logger.Error("error marshalling chain config", err)
		return nil, err
	}
	clbz, err := utils.MarshalJSONAny(m, client)
	if err != nil {
		logger.Error("error marshalling client config", err)
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
	} else if err := chain.Validate(); err != nil {
		return fmt.Errorf("invalid chain config: %v", err)
	}
	var prover ProverConfig
	if err := utils.UnmarshalJSONAny(m, &prover, cc.Prover); err != nil {
		return err
	} else if err := prover.Validate(); err != nil {
		return fmt.Errorf("invalid prover config: %v", err)
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

func SyncChainConfigsFromEvents(pathName string, msgIDsSrc, msgIDsDst []MsgID, src, dst *ProvableChain) error {
	if err := SyncChainConfigFromEvents(pathName, msgIDsSrc, src); err != nil {
		return err
	}
	if err := SyncChainConfigFromEvents(pathName, msgIDsDst, dst); err != nil {
		return err
	}
	return nil
}

func SyncChainConfigFromEvents(pathName string, msgIDs []MsgID, chain *ProvableChain) error {
	for _, msgID := range msgIDs {
		if msgID == nil {
			continue
		}
		msgRes, err := chain.Chain.GetMsgResult(msgID)
		if err != nil {
			return fmt.Errorf("failed to get message result: %v", err)
		} else if ok, failureReason := msgRes.Status(); !ok {
			return fmt.Errorf("msg(id=%v) execution failed: %v", msgID, failureReason)
		}

		for _, event := range msgRes.Events() {
			var id string
			var configID ConfigIDType
			switch event := event.(type) {
			case *EventGenerateClientIdentifier:
				configID = ConfigIDClient
				id = event.ID
			case *EventGenerateConnectionIdentifier:
				configID = ConfigIDConnection
				id = event.ID
			case *EventGenerateChannelIdentifier:
				configID = ConfigIDChannel
				id = event.ID
			}
			if id != "" {
				if err := config.UpdateConfigID(pathName, chain.ChainID(), configID, id); err != nil {
					return err
				}
			}
		}
	}

	return nil
}
