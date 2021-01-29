package corda

import (
	"fmt"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/simapp/params"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/datachainlab/relayer/core"
)

type Chain struct {
	config         ChainConfig
	pathEnd        *core.PathEnd
	encodingConfig params.EncodingConfig

	client *cordaIbcClient
}

func NewChain(config ChainConfig) *Chain {
	return &Chain{config: config}
}

var _ core.ChainI = (*Chain)(nil)

func (*Chain) ClientType() string {
	return "corda"
}

func (c *Chain) ChainID() string {
	return c.config.ChainId
}

func (c *Chain) ClientID() string {
	return c.pathEnd.ClientID
}

func (c *Chain) GetAddress() (sdk.AccAddress, error) {
	return []byte{}, nil
}

func (c *Chain) GetLatestLightHeight() (int64, error) {
	return 0, nil
}

func (c *Chain) Marshaler() codec.Marshaler {
	return c.encodingConfig.Marshaler
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

func (c *Chain) Update(key, value string) (core.ChainConfigI, error) {
	panic("not implemented error")
}

// MakeMsgCreateClient creates a CreateClientMsg to this chain
func (c *Chain) MakeMsgCreateClient(clientID string, dstHeader core.HeaderI, signer sdk.AccAddress) (sdk.Msg, error) {
	panic("not implemented error")
}

// CreateTrustedHeader creates ...
func (c *Chain) CreateTrustedHeader(dstChain core.ChainI, srcHeader core.HeaderI) (core.HeaderI, error) {
	return nil, nil
}

func (c *Chain) StartEventListener(dst core.ChainI, strategy core.StrategyI) {
	panic("not implemented error")
}

func (c *Chain) Init(homePath string, timeout time.Duration, debug bool) error {
	c.encodingConfig = makeEncodingConfig()
	if client, err := createCordaIbcClient(c.config.GrpcAddr); err != nil {
		return err
	} else {
		c.client = client
	}
	return nil
}

// errCantSetPath returns PartialEq, an error if the path doesn't set properly
func (c *Chain) errCantSetPath(err error) error {
	return fmt.Errorf("path on chain %s failed to set: %w", c.ChainID(), err)
}
