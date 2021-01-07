package fabric

func (c *Chain) PopulateWallet(certPath, privateKeyPath string) error {
	return c.gateway.PopulateWallet(c.homePath, c.config.MspId, certPath, privateKeyPath)
}
