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
	Global GlobalConfig             `yaml:"global" json:"global"`
	Chains []core.ChainProverConfig `yaml:"chains" json:"chains"`
	Paths  core.Paths               `yaml:"paths" json:"paths"`

	// cache
	chains Chains `yaml:"-" json:"-"`

	ConfigPath string `yaml:"-" json:"-"`
}

func defaultConfig(configPath string) Config {
	return Config{
		Global:     newDefaultGlobalConfig(),
		Chains:     []core.ChainProverConfig{},
		Paths:      core.Paths{},
		ConfigPath: configPath,
	}
}

// GlobalConfig describes any global relayer settings
type GlobalConfig struct {
	Timeout        string       `yaml:"timeout" json:"timeout"`
	LightCacheSize int          `yaml:"light-cache-size" json:"light-cache-size"`
	LoggerConfig   LoggerConfig `yaml:"logger" json:"logger"`
}

type LoggerConfig struct {
	Level  string `yaml:"level" json:"level"`
	Format string `yaml:"format" json:"format"`
	Output string `yaml:"output" json:"output"`
}

// newDefaultGlobalConfig returns a global config with defaults set
func newDefaultGlobalConfig() GlobalConfig {
	return GlobalConfig{
		Timeout:        "10s",
		LightCacheSize: 20,
		LoggerConfig: LoggerConfig{
			Level:  "DEBUG",
			Format: "json",
			Output: "stderr",
		},
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
func initChains(ctx *Context, homePath string, debug bool) error {
	to, err := time.ParseDuration(ctx.Config.Global.Timeout)
	if err != nil {
		return fmt.Errorf("did you remember to run 'rly config init' error:%w", err)
	}

	for _, chain := range ctx.Config.chains {
		if err := chain.Init(homePath, to, ctx.Codec, debug); err != nil {
			return fmt.Errorf("did you remember to run 'rly config init' error:%w", err)
		}
	}

	return nil
}

func (c *Config) UnmarshalConfig(homePath, configPath string) error {
	cfgPath := fmt.Sprintf("%s/%s", homePath, configPath)
	if _, err := os.Stat(cfgPath); err == nil {
		file, err := os.ReadFile(cfgPath)
		if err != nil {
			return err
		}
		// unmarshall them into the struct
		if err = json.Unmarshal(file, c); err != nil {
			return err
		}
	} else {
		defConfig := defaultConfig(cfgPath)
		*c = defConfig
	}
	c.ConfigPath = cfgPath
	return nil
}

func (ctx *Context) InitConfig(homePath string, debug bool) error {
	for _, c := range ctx.Config.Chains {
		if err := c.Init(ctx.Codec); err != nil {
			return err
		}
		chain, err := c.Build()
		if err != nil {
			return err
		}
		ctx.Config.chains = append(ctx.Config.chains, chain)
	}

	// ensure config has []*relayer.Chain used for all chain operations
	if err := initChains(ctx, homePath, debug); err != nil {
		return err
	}
	ctx.Config.InitCoreConfig()
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
