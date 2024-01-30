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

func (c CoreConfig) UpdateConfigID(pathName string, chainID string, configID core.ConfigIDType, id string) error {
	configPath, err := c.config.Paths.Get(pathName)
	if err != nil {
		return err
	}
	var pathEnd *core.PathEnd
	if chainID == configPath.Src.ChainID {
		pathEnd = configPath.Src
	}
	if chainID == configPath.Dst.ChainID {
		pathEnd = configPath.Dst
	}
	if pathEnd == nil {
		return fmt.Errorf("pathEnd is nil")
	}
	switch configID {
	case core.ConfigIDClient:
		pathEnd.ClientID = id
	case core.ConfigIDConnection:
		pathEnd.ConnectionID = id
	case core.ConfigIDChannel:
		pathEnd.ChannelID = id
	}
	if err := c.config.OverWriteConfig(); err != nil {
		return err
	}
	return nil
}
