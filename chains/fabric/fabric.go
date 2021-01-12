package fabric

import (
	"fmt"
	"path/filepath"

	"github.com/gogo/protobuf/proto"
	msppb "github.com/hyperledger/fabric-protos-go/msp"
	"github.com/hyperledger/fabric-sdk-go/pkg/gateway"
	"github.com/hyperledger/fabric/bccsp/factory"
	"github.com/hyperledger/fabric/msp"
)

func (c *Chain) Connect() error {
	return c.gateway.Connect(
		c.getWalletPath(),
		c.config.MspId,
		c.config.ConnectionProfilePath,
		c.config.Channel,
		c.config.ChaincodeId,
	)
}

func (c *Chain) Contract() *gateway.Contract {
	if c.gateway.Contract == nil {
		if err := c.Connect(); err != nil {
			panic(err)
		}
	}
	return c.gateway.Contract
}

type MSPConfig struct {
	MSPID  string
	Config msppb.MSPConfig
}

// get MSP Configs for Chain.IBCPolicies
func (c *Chain) GetLocalMspConfigs() ([]MSPConfig, error) {
	if len(c.config.IbcPolicies) != len(c.config.MspConfigPaths) {
		return nil, fmt.Errorf("IBCPolicies and MspConfigPaths must have the same length for now: %v != %v", len(c.config.IbcPolicies), len(c.config.MspConfigPaths))
	}
	res := []MSPConfig{}
	for i, path := range c.config.MspConfigPaths {
		mspId := c.config.IbcPolicies[i]
		bccspConfig := factory.GetDefaultOpts()
		mspConf, err := msp.GetLocalMspConfig(filepath.Clean(path), bccspConfig, mspId)
		if err != nil {
			return nil, err
		}
		if err := getVerifyingConfig(mspConf); err != nil {
			return nil, err
		}
		res = append(res, MSPConfig{MSPID: mspId, Config: *mspConf})
	}
	return res, nil
}

func (c *Chain) getSerializedIdentity(label string) (*msppb.SerializedIdentity, error) {
	creds, err := c.gateway.Wallet.Get(label)
	if err != nil {
		return nil, err
	}
	identity := creds.(*gateway.X509Identity)
	return &msppb.SerializedIdentity{
		Mspid:   identity.MspID,
		IdBytes: []byte(identity.Certificate()),
	}, nil
}

// remove SigningIdentity for verifying only purpose.
func getVerifyingConfig(mconf *msppb.MSPConfig) error {
	var conf msppb.FabricMSPConfig
	err := proto.Unmarshal(mconf.Config, &conf)
	if err != nil {
		return err
	}
	conf.SigningIdentity = nil
	confb, err := proto.Marshal(&conf)
	if err != nil {
		return err
	}
	mconf.Config = confb
	return nil
}
