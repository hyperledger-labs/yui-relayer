package fabric

import (
	"path/filepath"

	msppb "github.com/hyperledger/fabric-protos-go/msp"
	"github.com/hyperledger/fabric-sdk-go/pkg/gateway"
)

func (c *Chain) PopulateWallet(certPath, privateKeyPath string) error {
	return c.gateway.PopulateWallet(c.getWalletPath(), c.config.WalletLabel, certPath, privateKeyPath)
}

func (c *Chain) getWalletPath() string {
	walletPath := filepath.Join(c.homePath, "wallet")
	return filepath.Clean(walletPath)
}

func (c *Chain) GetSerializedIdentity(label string) (*msppb.SerializedIdentity, error) {
	w, err := c.gateway.Wallet.Get(label)
	if err != nil {
		return nil, err
	}
	id := w.(*gateway.X509Identity)
	return &msppb.SerializedIdentity{
		Mspid:   id.MspID,
		IdBytes: []byte(id.Certificate()),
	}, nil
}
