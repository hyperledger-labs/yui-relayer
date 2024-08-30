package config

import (
	"fmt"

	"github.com/hyperledger-labs/yui-relayer/core"
)

type CoreConfig struct {
	config *Config
}

var _ core.ConfigI = (*CoreConfig)(nil)

func initCoreConfig(c *Config) {
	config := &CoreConfig{
		config: c,
	}
	core.SetCoreConfig(config)
}

func (c CoreConfig) UpdatePathConfig(pathName string, chainID string, kv map[core.PathConfigKey]string) error {
	configPath, err := c.config.Paths.Get(pathName)
	if err != nil {
		return err
	}

	var pathEnd *core.PathEnd
	if chainID == configPath.Src.ChainID {
		pathEnd = configPath.Src
	} else if chainID == configPath.Dst.ChainID {
		pathEnd = configPath.Dst
	} else {
		return fmt.Errorf("pathEnd is nil")
	}

	for k, v := range kv {
		switch k {
		case core.PathConfigClientID:
			pathEnd.ClientID = v
		case core.PathConfigConnectionID:
			pathEnd.ConnectionID = v
		case core.PathConfigChannelID:
			pathEnd.ChannelID = v
		case core.PathConfigOrder:
			pathEnd.Order = v
		case core.PathConfigVersion:
			pathEnd.Version = v
		default:
			panic(fmt.Sprintf("unexpected path config key: %s", k))
		}
	}

	return c.config.OverWriteConfig()
}
