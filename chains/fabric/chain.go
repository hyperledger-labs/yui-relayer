package fabric

import (
	"fmt"
	"os"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/hyperledger-labs/yui-fabric-ibc/x/auth/types"
	"github.com/hyperledger-labs/yui-relayer/core"
	"github.com/tendermint/tendermint/libs/log"
)

type Chain struct {
	config ChainConfig

	pathEnd  *core.PathEnd
	homePath string

	codec   codec.ProtoCodecMarshaler
	gateway FabricGateway
	logger  log.Logger
}

func NewChain(config ChainConfig) *Chain {
	return &Chain{config: config}
}

var _ core.ChainI = (*Chain)(nil)

func (c *Chain) Init(homePath string, timeout time.Duration, codec codec.ProtoCodecMarshaler, debug bool) error {
	c.homePath = homePath
	c.codec = codec
	c.logger = log.NewTMLogger(log.NewSyncWriter(os.Stdout))
	return nil
}

// QueryLatestHeight queries the chain for the latest height and returns it
func (c *Chain) GetLatestHeight() (int64, error) {
	return -1, nil
}

func (c *Chain) ChainID() string {
	return c.config.ChainId
}

func (c *Chain) ClientID() string {
	return c.pathEnd.ClientID
}

func (c *Chain) Config() ChainConfig {
	return c.config
}

func (c *Chain) Codec() codec.ProtoCodecMarshaler {
	return c.codec
}

// GetAddress returns the sdk.AccAddress associated with the configred key
func (c *Chain) GetAddress() (sdk.AccAddress, error) {
	if c.gateway.Contract == nil {
		if err := c.Connect(); err != nil {
			return nil, err
		}
	}

	sid, err := c.getSerializedIdentity(c.config.WalletLabel)
	if err != nil {
		return nil, err
	}
	return authtypes.MakeCreatorAddressWithSerializedIdentity(sid)
}

func (c *Chain) SetPath(p *core.PathEnd) error {
	err := p.Validate()
	if err != nil {
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

// errCantSetPath returns an error if the path doesn't set properly
func (c *Chain) errCantSetPath(err error) error {
	return fmt.Errorf("path on chain %s failed to set: %w", c.ChainID(), err)
}
