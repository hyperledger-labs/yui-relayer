package fabric

import "path/filepath"

func (c *Chain) PopulateWallet(certPath, privateKeyPath string) error {
	return c.gateway.PopulateWallet(c.getWalletPath(), c.config.MspId, certPath, privateKeyPath)
}

func (c *Chain) getWalletPath() string {
	walletPath := filepath.Join(c.homePath, "wallet")
	return filepath.Clean(walletPath)
}
