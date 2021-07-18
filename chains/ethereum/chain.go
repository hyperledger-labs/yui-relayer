package ethereum

import (
	fmt "fmt"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	transfertypes "github.com/cosmos/ibc-go/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	conntypes "github.com/cosmos/ibc-go/modules/core/03-connection/types"
	chantypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
	ibcexported "github.com/cosmos/ibc-go/modules/core/exported"
	"github.com/hyperledger-labs/yui-relayer/core"
)

type Chain struct {
	config ChainConfig

	PathEnd *core.PathEnd `yaml:"-" json:"-"`
}

var _ core.ChainI = (*Chain)(nil)

func NewChain(config ChainConfig) *Chain {
	return &Chain{
		config: config,
	}
}

// ChainID returns ID of the chain
func (c *Chain) ChainID() string {
	return c.config.ChainId
}

// GetLatestHeight gets the chain for the latest height and returns it
func (c *Chain) GetLatestHeight() (int64, error) {
	panic("not implemented") // TODO: Implement
}

// GetAddress returns the address of relayer
func (c *Chain) GetAddress() (sdk.AccAddress, error) {
	panic("not implemented") // TODO: Implement
}

// Marshaler returns the marshaler
func (c *Chain) Marshaler() codec.Codec {
	panic("not implemented") // TODO: Implement
}

// SetPath sets the path and validates the identifiers
func (c *Chain) SetPath(p *core.PathEnd) error {
	err := p.Validate()
	if err != nil {
		return c.ErrCantSetPath(err)
	}
	c.PathEnd = p
	return nil
}

// ErrCantSetPath returns an error if the path doesn't set properly
func (c *Chain) ErrCantSetPath(err error) error {
	return fmt.Errorf("path on chain %s failed to set: %w", c.ChainID(), err)
}

func (c *Chain) Path() *core.PathEnd {
	return c.PathEnd
}

// SendMsgs sends msgs to the chain
func (c *Chain) SendMsgs(msgs []sdk.Msg) ([]byte, error) {
	panic("not implemented") // TODO: Implement
}

// Send sends msgs to the chain and logging a result of it
// It returns a boolean value whether the result is success
func (c *Chain) Send(msgs []sdk.Msg) bool {
	panic("not implemented") // TODO: Implement
}

// StartEventListener ...
func (c *Chain) StartEventListener(dst core.ChainI, strategy core.StrategyI) {
	panic("not implemented") // TODO: Implement
}

// Init ...
func (c *Chain) Init(homePath string, timeout time.Duration, debug bool) error {
	panic("not implemented") // TODO: Implement
}

// QueryClientConsensusState retrevies the latest consensus state for a client in state at a given height
func (c *Chain) QueryClientConsensusState(height int64, dstClientConsHeight ibcexported.Height) (*clienttypes.QueryConsensusStateResponse, error) {
	panic("not implemented") // TODO: Implement
}

// QueryClientState returns the client state of dst chain
// height represents the height of dst chain
func (c *Chain) QueryClientState(height int64) (*clienttypes.QueryClientStateResponse, error) {
	panic("not implemented") // TODO: Implement
}

// QueryConnection returns the remote end of a given connection
func (c *Chain) QueryConnection(height int64) (*conntypes.QueryConnectionResponse, error) {
	panic("not implemented") // TODO: Implement
}

// QueryChannel returns the channel associated with a channelID
func (c *Chain) QueryChannel(height int64) (chanRes *chantypes.QueryChannelResponse, err error) {
	panic("not implemented") // TODO: Implement
}

// QueryPacketCommitment returns the packet commitment corresponding to a given sequence
func (c *Chain) QueryPacketCommitment(height int64, seq uint64) (comRes *chantypes.QueryPacketCommitmentResponse, err error) {
	panic("not implemented") // TODO: Implement
}

// QueryPacketAcknowledgementCommitment returns the acknowledgement corresponding to a given sequence
func (c *Chain) QueryPacketAcknowledgementCommitment(height int64, seq uint64) (ackRes *chantypes.QueryPacketAcknowledgementResponse, err error) {
	panic("not implemented") // TODO: Implement
}

// QueryPacketCommitments returns an array of packet commitments
func (c *Chain) QueryPacketCommitments(offset uint64, limit uint64, height int64) (comRes *chantypes.QueryPacketCommitmentsResponse, err error) {
	panic("not implemented") // TODO: Implement
}

// QueryUnrecievedPackets returns a list of unrelayed packet commitments
func (c *Chain) QueryUnrecievedPackets(height int64, seqs []uint64) ([]uint64, error) {
	panic("not implemented") // TODO: Implement
}

// QueryPacketAcknowledgements returns an array of packet acks
func (c *Chain) QueryPacketAcknowledgements(offset uint64, limit uint64, height int64) (comRes *chantypes.QueryPacketAcknowledgementsResponse, err error) {
	panic("not implemented") // TODO: Implement
}

// QueryUnrecievedAcknowledgements returns a list of unrelayed packet acks
func (c *Chain) QueryUnrecievedAcknowledgements(height int64, seqs []uint64) ([]uint64, error) {
	panic("not implemented") // TODO: Implement
}

// QueryPacket returns the packet corresponding to a sequence
func (c *Chain) QueryPacket(height int64, sequence uint64) (*chantypes.Packet, error) {
	panic("not implemented") // TODO: Implement
}

// QueryPacketAcknowledgement returns the acknowledgement corresponding to a sequence
func (c *Chain) QueryPacketAcknowledgement(height int64, sequence uint64) ([]byte, error) {
	panic("not implemented") // TODO: Implement
}

// QueryBalance returns the amount of coins in the relayer account
func (c *Chain) QueryBalance(address sdk.AccAddress) (sdk.Coins, error) {
	panic("not implemented") // TODO: Implement
}

// QueryDenomTraces returns all the denom traces from a given chain
func (c *Chain) QueryDenomTraces(offset uint64, limit uint64, height int64) (*transfertypes.QueryDenomTracesResponse, error) {
	panic("not implemented") // TODO: Implement
}
