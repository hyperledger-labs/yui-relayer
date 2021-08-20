package corda

import (
	"context"
	"fmt"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	cordatypes "github.com/hyperledger-labs/yui-corda-ibc/go/x/ibc/light-clients/xx-corda/types"
	"github.com/hyperledger-labs/yui-relayer/core"
)

type Chain struct {
	config  ChainConfig
	pathEnd *core.PathEnd
	codec   codec.ProtoCodecMarshaler

	client         *cordaIbcClient
	bankNodeClient *cordaIbcClient
}

func NewChain(config ChainConfig) *Chain {
	return &Chain{config: config}
}

var _ core.ChainI = (*Chain)(nil)

func (c *Chain) ChainID() string {
	return c.config.ChainId
}

func (c *Chain) ClientID() string {
	return c.pathEnd.ClientID
}

func (c *Chain) GetAddress() (sdk.AccAddress, error) {
	req := cordatypes.AddressFromNameRequest{
		Name:       c.config.PartyName,
		ExactMatch: false,
	}
	resp, err := c.client.node.AddressFromName(context.TODO(), &req)
	if err != nil {
		return nil, err
	}
	addr, err := sdk.AccAddressFromBech32(resp.Address)
	if err != nil {
		return nil, err
	}
	return addr, nil
}

func (c *Chain) Codec() codec.ProtoCodecMarshaler {
	return c.codec
}

func (c *Chain) SetPath(p *core.PathEnd) error {
	if err := p.Validate(); err != nil {
		return c.errCantSetPath(err)
	}
	c.pathEnd = p
	return nil
}

func (c *Chain) Path() *core.PathEnd {
	return c.pathEnd
}

func (c *Chain) StartEventListener(dst core.ChainI, strategy core.StrategyI) {
	panic("not implemented error")
}

func (c *Chain) Init(homePath string, timeout time.Duration, codec codec.ProtoCodecMarshaler, debug bool) error {
	if client, err := createCordaIbcClient(c.config.GrpcAddr); err != nil {
		return err
	} else {
		c.client = client
	}
	if client, err := createCordaIbcClient(c.config.BankGrpcAddr); err != nil {
		return err
	} else {
		c.bankNodeClient = client
	}
	c.codec = codec
	return nil
}

// errCantSetPath returns PartialEq, an error if the path doesn't set properly
func (c *Chain) errCantSetPath(err error) error {
	return fmt.Errorf("path on chain %s failed to set: %w", c.ChainID(), err)
}
