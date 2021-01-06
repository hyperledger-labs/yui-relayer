package config

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	relayercmd "github.com/cosmos/relayer/cmd"
	"github.com/cosmos/relayer/relayer"
	"github.com/datachainlab/relayer/core"
)

type Config struct {
	Global relayercmd.GlobalConfig `yaml:"global" json:"global"`
	Chains []json.RawMessage       `yaml:"chains" json:"chains"`
	Paths  relayer.Paths           `yaml:"paths" json:"paths"`

	// cache
	chains Chains `yaml:"-" json:"-"`
}

func DefaultConfig() Config {
	return Config{
		Global: newDefaultGlobalConfig(),
		Chains: []json.RawMessage{},
		Paths:  relayer.Paths{},
	}
}

// newDefaultGlobalConfig returns a global config with defaults set
func newDefaultGlobalConfig() relayercmd.GlobalConfig {
	return relayercmd.GlobalConfig{
		Timeout:        "10s",
		LightCacheSize: 20,
	}
}

func (c *Config) GetChain(chainID string) (core.ChainI, error) {
	for _, c := range c.chains {
		if c.ChainID() == chainID {
			return c, nil
		}
	}
	return nil, fmt.Errorf("chainID '%v' not found", chainID)
}

// AddChain adds an additional chain to the config
func (c *Config) AddChain(m codec.JSONMarshaler, cconfig ChainConfigI) error {
	chain := cconfig.GetChain()
	_, err := c.GetChain(chain.ChainID())
	if err == nil {
		return fmt.Errorf("chain with ID %s already exists in config", chain.ChainID())
	}
	bz, err := MarshalJSONAny(m, cconfig)
	if err != nil {
		return err
	}
	c.Chains = append(c.Chains, bz)
	c.chains = append(c.chains, chain)
	return nil
}

// AddPath adds an additional path to the config
func (c *Config) AddPath(name string, path *relayer.Path) (err error) {
	return c.Paths.Add(name, path)
}

// ChainsFromPath takes the path name and returns the properly configured chains
func (c *Config) ChainsFromPath(path string) (map[string]core.ChainI, string, string, error) {
	pth, err := c.Paths.Get(path)
	if err != nil {
		return nil, "", "", err
	}

	src, dst := pth.Src.ChainID, pth.Dst.ChainID
	chains, err := c.chains.Gets(src, dst)
	if err != nil {
		return nil, "", "", err
	}

	if err = chains[src].SetPath(pth.Src); err != nil {
		return nil, "", "", err
	}
	if err = chains[dst].SetPath(pth.Dst); err != nil {
		return nil, "", "", err
	}

	return chains, src, dst, nil
}

// Called to initialize the relayer.Chain types on Config
func InitChains(c *Config, homePath string, debug bool) error {
	to, err := time.ParseDuration(c.Global.Timeout)
	if err != nil {
		return fmt.Errorf("did you remember to run 'rly config init' error:%w", err)
	}

	for _, chain := range c.chains {
		if err := chain.Init(homePath, to, debug); err != nil {
			return fmt.Errorf("did you remember to run 'rly config init' error:%w", err)
		}
	}

	return nil
}

type ConfigManager interface {
	Get() *Config
	Set(Config)
}
