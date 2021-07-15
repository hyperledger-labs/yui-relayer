package fabric

import (
	"github.com/hyperledger/fabric-sdk-go/pkg/gateway"
)

func (c *Chain) Connect() error {
	return c.gateway.Connect(
		c.getWalletPath(),
		c.config.WalletLabel,
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
