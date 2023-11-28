package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path"

	"github.com/hyperledger-labs/yui-relayer/core"
)

type CoreConfig struct {
	config   Config
	pathName string
}

var _ core.ConfigI = (*CoreConfig)(nil)

func initCoreConfig(c Config, pathName string) {
	config := &CoreConfig{
		config:   c,
		pathName: pathName,
	}
	core.CoreConfig(config)
}

func (c CoreConfig) UpdateConfigID(chainID string, key core.PathEndName, ID string) error {
	configPath, err := c.config.Paths.Get(c.pathName)
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
	switch key {
	case core.PathEndNameClient:
		pathEnd.ClientID = ID
	case core.PathEndNameConnection:
		pathEnd.ConnectionID = ID
	case core.PathEndNameChannel:
		pathEnd.ChannelID = ID
	}
	configData, err := json.Marshal(c.config)
	if err != nil {
		return err
	}
	cfgPath := path.Join(c.config.Global.HomePath, "config", "config.yaml")
	if err := os.WriteFile(cfgPath, configData, 0600); err != nil {
		return err
	}
	return nil
}
