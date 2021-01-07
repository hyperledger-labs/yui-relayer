package fabric

import "github.com/hyperledger/fabric-sdk-go/pkg/gateway"

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
	return c.gateway.Contract
}
