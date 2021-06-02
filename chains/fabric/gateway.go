package fabric

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/core"
	"github.com/hyperledger/fabric-sdk-go/pkg/core/config"
	"github.com/hyperledger/fabric-sdk-go/pkg/gateway"
)

type FabricGateway struct {
	ConfigProvider core.ConfigProvider `yaml:"-" json:"-"`
	Wallet         *gateway.Wallet     `yaml:"-" json:"-"`
	Gateway        *gateway.Gateway    `yaml:"-" json:"-"`
	Network        *gateway.Network    `yaml:"-" json:"-"`
	Contract       *gateway.Contract   `yaml:"-" json:"-"`
}

func (gw *FabricGateway) PopulateWallet(walletPath, mspID, certPath, privateKeyPath string) error {
	if err := gw.fileSystemWallet(walletPath); err != nil {
		return err
	}

	if gw.Wallet.Exists(mspID) {
		return nil
	}

	// read the certificate pem
	cert, err := ioutil.ReadFile(filepath.Clean(certPath))
	if err != nil {
		return err
	}

	key, err := ioutil.ReadFile(filepath.Clean(privateKeyPath))
	if err != nil {
		return err
	}

	identity := gateway.NewX509Identity(mspID, string(cert), string(key))

	err = gw.Wallet.Put(mspID, identity)
	if err != nil {
		return err
	}

	return nil
}

func (gw *FabricGateway) Connect(
	walletPath string,
	mspID string,
	ccpPath string,
	channel string,
	chaincodeID string,
) error {
	var err error
	if err = gw.fileSystemWallet(walletPath); err != nil {
		return err
	}

	if !gw.Wallet.Exists(mspID) {
		return fmt.Errorf("not exists wallet: %v.id", mspID)
	}

	gw.Gateway, err = gateway.Connect(
		gateway.WithConfig(config.FromFile(filepath.Clean(ccpPath))),
		gateway.WithIdentity(gw.Wallet, mspID),
	)
	if err != nil {
		return err
	}

	gw.Network, err = gw.Gateway.GetNetwork(channel)
	if err != nil {
		return err
	}

	gw.Contract = gw.Network.GetContract(chaincodeID)
	if gw.Contract == nil {
		return err
	}

	return nil
}

func (gw *FabricGateway) fileSystemWallet(walletPath string) error {
	var err error
	if gw.Wallet, err = gateway.NewFileSystemWallet(walletPath); err != nil {
		return err
	}
	return nil
}
