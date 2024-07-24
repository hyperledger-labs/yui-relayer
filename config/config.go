package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/hyperledger-labs/yui-relayer/core"
)

type Config struct {
	Chains []core.ChainProverConfig `yaml:"chains" json:"chains"`
	Paths  core.Paths               `yaml:"paths" json:"paths"`

	// cache
	chains Chains `yaml:"-" json:"-"`

	ConfigPath string `yaml:"-" json:"-"`
}

func defaultConfig(configPath string) Config {
	return Config{
		Chains:     []core.ChainProverConfig{},
		Paths:      core.Paths{},
		ConfigPath: configPath,
	}
}

func (c *Config) InitCoreConfig() {
	initCoreConfig(c)
}

func (c *Config) GetChain(chainID string) (*core.ProvableChain, error) {
	return c.chains.Get(chainID)
}

func (c *Config) GetChains(chainIDs ...string) (map[string]*core.ProvableChain, error) {
	return c.chains.Gets(chainIDs...)
}

// AddChain adds an additional chain to the config
func (c *Config) AddChain(m codec.JSONCodec, config core.ChainProverConfig) error {
	chain, err := config.Build()
	if err != nil {
		return err
	}
	if _, err := c.GetChain(chain.ChainID()); err == nil {
		return fmt.Errorf("chain with ID %s already exists in config", chain.ChainID())
	}
	c.Chains = append(c.Chains, config)
	c.chains = append(c.chains, chain)
	return nil
}

// AddPath adds an additional path to the config
func (c *Config) AddPath(name string, path *core.Path) (err error) {
	return c.Paths.Add(name, path)
}

// DeleteChain removes a chain from the config
func (c *Config) DeleteChain(chain string) *Config {
	var newChains []*core.ProvableChain
	var newChainConfigs []core.ChainProverConfig
	for i, ch := range c.chains {
		if ch.ChainID() != chain {
			newChains = append(newChains, ch)
			newChainConfigs = append(newChainConfigs, c.Chains[i])
		}
	}
	c.Chains = newChainConfigs
	c.chains = newChains
	return c
}

// ChainsFromPath takes the path name and returns the properly configured chains
func (c *Config) ChainsFromPath(path string) (map[string]*core.ProvableChain, string, string, error) {

	pth, err := c.Paths.Get(path)
	if err != nil {
		return nil, "", "", err
	}

	src, dst := pth.Src.ChainID, pth.Dst.ChainID
	chains, err := c.chains.Gets(src, dst)
	if err != nil {
		return nil, "", "", err
	}

	if err = chains[src].SetRelayInfo(pth.Src, chains[dst], pth.Dst); err != nil {
		return nil, "", "", err
	}
	if err = chains[dst].SetRelayInfo(pth.Dst, chains[src], pth.Src); err != nil {
		return nil, "", "", err
	}

	return chains, src, dst, nil
}

// Called to initialize the relayer.Chain types on Config
func InitChains(ctx *Context, homePath string, debug bool, timeout time.Duration) error {
	for _, chain := range ctx.Config.chains {
		if err := chain.Init(homePath, timeout, ctx.Codec, debug); err != nil {
			return fmt.Errorf("did you remember to run 'rly config init' error:%w", err)
		}
	}

	return nil
}

func (c *Config) InitConfig(ctx *Context, homePath, configPath string, debug bool, timeout time.Duration) error {
	cfgPath := fmt.Sprintf("%s/%s", homePath, configPath)
	c.ConfigPath = cfgPath
	if _, err := os.Stat(cfgPath); err == nil {
		file, err := os.ReadFile(cfgPath)
		if err != nil {
			return err
		}
		// unmarshall them into the struct
		if err = UnmarshalJSON(ctx.Codec, file, c); err != nil {
			return err
		}
		// ensure config has []*relayer.Chain used for all chain operations
		if err = InitChains(ctx, homePath, debug, timeout); err != nil {
			return err
		}
	} else {
		defConfig := defaultConfig(cfgPath)
		c = &defConfig
	}
	c.InitCoreConfig()
	ctx.Config = c
	return nil
}

func (c *Config) CreateConfig() error {
	cfgPath := c.ConfigPath
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		dirPath := filepath.Dir(cfgPath)
		if err := os.MkdirAll(dirPath, os.ModePerm); err != nil {
			return err
		}
		f, err := os.Create(cfgPath)
		if err != nil {
			return err
		}
		defer f.Close()
		if _, err = f.Write(defaultConfigBytes(c.ConfigPath)); err != nil {
			return err
		}
		return nil
	} else if err != nil {
		return err
	}
	return nil
}

func (c *Config) OverWriteConfig() error {
	configData, err := json.Marshal(c)
	if err != nil {
		return err
	}
	if err := os.WriteFile(c.ConfigPath, configData, 0600); err != nil {
		return err
	}
	return nil
}

func defaultConfigBytes(configPath string) []byte {
	bz, err := json.Marshal(defaultConfig(configPath))
	if err != nil {
		panic(err)
	}
	return bz
}
