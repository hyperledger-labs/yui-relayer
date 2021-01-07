package fabric

import (
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/datachainlab/relayer/core"
)

type Chain struct {
	config ChainConfig

	pathEnd  *core.PathEnd
	homePath string

	gateway *FabricGateway
}

func NewChain(config ChainConfig) *Chain {
	return &Chain{config: config}
}

var _ core.ChainI = (*Chain)(nil)

func (c *Chain) ClientType() string {
	return "fabric"
}

func (c *Chain) ChainID() string {
	return c.config.ChainId
}

func (c *Chain) ClientID() string {
	return c.pathEnd.ClientID
}

// GetAddress returns the sdk.AccAddress associated with the configred key
func (c *Chain) GetAddress() (sdk.AccAddress, error) {
	// FIXME returns an address correctly
	return sdk.AccAddress("dummy"), nil
}

func (c *Chain) SetPath(p *core.PathEnd) error {
	err := p.Validate()
	if err != nil {
		return c.errCantSetPath(err)
	}
	c.pathEnd = p
	return nil
}

func (c *Chain) Update(key, value string) (core.ChainConfigI, error) {
	panic("not implemented error")
	return &c.config, nil
}

func (c *Chain) Init(homePath string, timeout time.Duration, debug bool) error {
	c.homePath = homePath
	return nil
}

func (c *Chain) StartEventListener(dst core.ChainI, strategy core.StrategyI) {
	panic("not implemented error")
}

func (c *Chain) QueryLatestHeader() (core.HeaderI, error) {
	panic("not implemented error")
}

func (c *Chain) SendMsgs(msgs []sdk.Msg) ([]byte, error) {
	panic("not implemented error")
}

func (c *Chain) Send(msgs []sdk.Msg) bool {
	panic("not implemented error")
}

// errCantSetPath returns an error if the path doesn't set properly
func (c *Chain) errCantSetPath(err error) error {
	return fmt.Errorf("path on chain %s failed to set: %w", c.ChainID(), err)
}
