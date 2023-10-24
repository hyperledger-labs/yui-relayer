package core

import (
	"encoding/json"
	"fmt"
	"os"

	retry "github.com/avast/retry-go"
)

type Config struct {
	Global GlobalConfig        `yaml:"global" json:"global"`
	Chains []ChainProverConfig `yaml:"chains" json:"chains"`
	Paths  Paths               `yaml:"paths" json:"paths"`
}

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

func SyncChainConfigFromEvents(msgIDsSrc, msgIDsDst []MsgID, src, dst *ProvableChain, key string) error {
	if err := ProcessPathMsgIDs(msgIDsSrc, src, key); err != nil {
		return err
	}
	if err := ProcessPathMsgIDs(msgIDsDst, dst, key); err != nil {
		return err
	}
	return nil
}

func ProcessPathMsgIDs(msgIDs []MsgID, chain *ProvableChain, key string) error {
	for _, msgID := range msgIDs {
		msgRes, err := chain.Chain.GetMsgResult(msgID)
		if err != nil {
			return retry.Unrecoverable(fmt.Errorf("failed to get message result: %v", err))
		} else if ok, failureReason := msgRes.Status(); !ok {
			return retry.Unrecoverable(fmt.Errorf("msg(id=%v) execution failed: %v", msgID, failureReason))
		}
		for _, event := range msgRes.Events() {
			switch key {
			case "client":
				if clientIdentifier, ok := event.(*EventGenerateClientIdentifier); ok {
					id := clientIdentifier.ID
					if err := UpdateConfigID(chain, id, key); err != nil {
						return err
					}
					chain.Chain.Path().ClientID = id
				}
			case "connection":
				if connectionIdentifier, ok := event.(*EventGenerateConnectionIdentifier); ok {
					id := connectionIdentifier.ID
					if err := UpdateConfigID(chain, id, key); err != nil {
						return err
					}
					chain.Chain.Path().ConnectionID = id
				}
			case "channel":
				if channelIdentifier, ok := event.(*EventGenerateChannelIdentifier); ok {
					id := channelIdentifier.ID
					if err := UpdateConfigID(chain, id, key); err != nil {
						return err
					}
					chain.Chain.Path().ChannelID = id
				}
			}
		}
	}
	return nil
}

func UpdateConfigID(chain *ProvableChain, ID, key string) error {
	configFile := "/Users/dongri.jin/.yui-relayer/config/config.yaml"
	data, err := os.ReadFile(configFile)
	if err != nil {
		return err
	}
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return err
	}
	configPath, err := config.Paths.Get("ibc01")
	if err != nil {
		return err
	}
	var pathEnd *PathEnd
	if chain.Path().ChainID == configPath.Src.ChainID {
		pathEnd = configPath.Src
	}
	if chain.Path().ChainID == configPath.Dst.ChainID {
		pathEnd = configPath.Dst
	}
	if pathEnd == nil {
		return fmt.Errorf("pathEnd is nil")
	}
	switch key {
	case "client":
		pathEnd.ClientID = ID
	case "connection":
		pathEnd.ConnectionID = ID
	case "channel":
		pathEnd.ChannelID = ID
	}
	configData, err := json.Marshal(config)
	if err != nil {
		return err
	}
	if err := os.WriteFile(configFile, configData, 0666); err != nil {
		return err
	}
	return nil
}
