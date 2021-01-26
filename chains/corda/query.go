package corda

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	transfertypes "github.com/cosmos/cosmos-sdk/x/ibc/applications/transfer/types"
	clienttypes "github.com/cosmos/cosmos-sdk/x/ibc/core/02-client/types"
	conntypes "github.com/cosmos/cosmos-sdk/x/ibc/core/03-connection/types"
	chantypes "github.com/cosmos/cosmos-sdk/x/ibc/core/04-channel/types"
	ibcexported "github.com/cosmos/cosmos-sdk/x/ibc/core/exported"
	"github.com/datachainlab/relayer/core"
)

// QueryLatestHeight queries the chain for the latest height and returns it
func (c *Chain) QueryLatestHeight() (int64, error) {
	panic("not implemented error")
}

// QueryLatestHeader returns the latest header from the chain
func (c *Chain) QueryLatestHeader() (out core.HeaderI, err error) {
	panic("not implemented error")
}

// QueryClientConsensusState retrevies the latest consensus state for a client in state at a given height
func (c *Chain) QueryClientConsensusState(height int64, dstClientConsHeight ibcexported.Height, prove bool) (*clienttypes.QueryConsensusStateResponse, error) {
	panic("not implemented error")
}

// height represents the height of src chain
func (c *Chain) QueryClientState(height int64, prove bool) (*clienttypes.QueryClientStateResponse, error) {
	panic("not implemented error")
}

// QueryConnection returns the remote end of a given connection
func (c *Chain) QueryConnection(height int64, prove bool) (*conntypes.QueryConnectionResponse, error) {
	panic("not implemented error")
}

// QueryChannel returns the channel associated with a channelID
func (c *Chain) QueryChannel(height int64, prove bool) (chanRes *chantypes.QueryChannelResponse, err error) {
	panic("not implemented error")
}

// QueryBalance returns the amount of coins in the relayer account
func (c *Chain) QueryBalance(address sdk.AccAddress) (sdk.Coins, error) {
	panic("not implemented error")
}

// QueryDenomTraces returns all the denom traces from a given chain
func (c *Chain) QueryDenomTraces(offset, limit uint64, height int64) (*transfertypes.QueryDenomTracesResponse, error) {
	panic("not implemented error")
}

// QueryPacketCommitment returns the packet commitment proof at a given height
func (c *Chain) QueryPacketCommitment(height int64, seq uint64) (comRes *chantypes.QueryPacketCommitmentResponse, err error) {
	panic("not implemented error")
}

// QueryPacketCommitments returns an array of packet commitments
func (c *Chain) QueryPacketCommitments(offset, limit, height uint64) (comRes *chantypes.QueryPacketCommitmentsResponse, err error) {
	panic("not implemented error")
}

// QueryUnrecievedPackets returns a list of unrelayed packet commitments
func (c *Chain) QueryUnrecievedPackets(height uint64, seqs []uint64) ([]uint64, error) {
	panic("not implemented error")
}

// QueryPacketAcknowledgements returns an array of packet acks
func (c *Chain) QueryPacketAcknowledgements(offset, limit, height uint64) (comRes *chantypes.QueryPacketAcknowledgementsResponse, err error) {
	panic("not implemented error")
}

// QueryUnrecievedAcknowledgements returns a list of unrelayed packet acks
func (c *Chain) QueryUnrecievedAcknowledgements(height uint64, seqs []uint64) ([]uint64, error) {
	panic("not implemented error")
}

// QueryPacketAcknowledgementCommitment returns the packet ack proof at a given height
func (c *Chain) QueryPacketAcknowledgementCommitment(height int64, seq uint64) (ackRes *chantypes.QueryPacketAcknowledgementResponse, err error) {
	panic("not implemented error")
}

// QueryPacket returns a packet corresponds to a given sequence
func (c *Chain) QueryPacket(height int64, sequence uint64) (*chantypes.Packet, error) {
	panic("not implemented error")
}

func (c *Chain) QueryPacketAcknowledgement(height int64, sequence uint64) ([]byte, error) {
	panic("not implemented error")
}
