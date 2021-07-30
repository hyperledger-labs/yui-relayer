package corda

import (
	"fmt"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/hyperledger-labs/yui-relayer/core"
)

type Chain struct {
	config  ChainConfig
	pathEnd *core.PathEnd
	codec   codec.ProtoCodecMarshaler

	client *cordaIbcClient
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
	return make([]byte, 20), nil
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
	c.codec = codec
	return nil
}

// errCantSetPath returns PartialEq, an error if the path doesn't set properly
func (c *Chain) errCantSetPath(err error) error {
	return fmt.Errorf("path on chain %s failed to set: %w", c.ChainID(), err)
}
