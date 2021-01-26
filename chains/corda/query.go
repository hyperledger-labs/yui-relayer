package corda

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	transfertypes "github.com/cosmos/cosmos-sdk/x/ibc/applications/transfer/types"
	clienttypes "github.com/cosmos/cosmos-sdk/x/ibc/core/02-client/types"
	conntypes "github.com/cosmos/cosmos-sdk/x/ibc/core/03-connection/types"
	chantypes "github.com/cosmos/cosmos-sdk/x/ibc/core/04-channel/types"
	ibcexported "github.com/cosmos/cosmos-sdk/x/ibc/core/exported"
	"github.com/datachainlab/relayer/core"
)

// QueryLatestHeight queries the chain for the latest height and returns it
func (c *Chain) QueryLatestHeight() (int64, error) {
	return 0, nil
}

// QueryLatestHeader returns the latest header from the chain
func (c *Chain) QueryLatestHeader() (out core.HeaderI, err error) {
	return &cordaHeader{}, nil
}

// QueryClientConsensusState retrevies the latest consensus state for a client in state at a given height
func (c *Chain) QueryClientConsensusState(height int64, dstClientConsHeight ibcexported.Height, prove bool) (*clienttypes.QueryConsensusStateResponse, error) {
	return c.client.clientQuery.ConsensusState(
		context.TODO(),
		&clienttypes.QueryConsensusStateRequest{
			ClientId:      c.pathEnd.ClientID,
			VersionNumber: dstClientConsHeight.GetVersionNumber(),
			VersionHeight: dstClientConsHeight.GetVersionHeight(),
			LatestHeight:  false,
		},
	)
}

// height represents the height of src chain
func (c *Chain) QueryClientState(height int64, prove bool) (*clienttypes.QueryClientStateResponse, error) {
	return c.client.clientQuery.ClientState(
		context.TODO(),
		&clienttypes.QueryClientStateRequest{
			ClientId: c.pathEnd.ClientID,
		},
	)
}

// QueryConnection returns the remote end of a given connection
func (c *Chain) QueryConnection(height int64, prove bool) (*conntypes.QueryConnectionResponse, error) {
	return c.client.connQuery.Connection(
		context.TODO(),
		&conntypes.QueryConnectionRequest{
			ConnectionId: c.pathEnd.ConnectionID,
		},
	)
}

// QueryChannel returns the channel associated with a channelID
func (c *Chain) QueryChannel(height int64, prove bool) (chanRes *chantypes.QueryChannelResponse, err error) {
	return c.client.chanQuery.Channel(
		context.TODO(),
		&chantypes.QueryChannelRequest{
			PortId:    c.pathEnd.PortID,
			ChannelId: c.pathEnd.ChannelID,
		},
	)
}

// QueryBalance returns the amount of coins in the relayer account
func (c *Chain) QueryBalance(address sdk.AccAddress) (sdk.Coins, error) {
	panic("not implemented error")
}

// QueryDenomTraces returns all the denom traces from a given chain
func (c *Chain) QueryDenomTraces(offset, limit uint64, height int64) (*transfertypes.QueryDenomTracesResponse, error) {
	return c.client.transferQuery.DenomTraces(
		context.TODO(),
		&transfertypes.QueryDenomTracesRequest{Pagination: makePagination(offset, limit)},
	)
}

// QueryPacketCommitment returns the packet commitment proof at a given height
func (c *Chain) QueryPacketCommitment(height int64, seq uint64) (comRes *chantypes.QueryPacketCommitmentResponse, err error) {
	return c.client.chanQuery.PacketCommitment(
		context.TODO(),
		&chantypes.QueryPacketCommitmentRequest{
			PortId:    c.pathEnd.PortID,
			ChannelId: c.pathEnd.ChannelID,
			Sequence:  seq,
		},
	)
}

// QueryPacketCommitments returns an array of packet commitments
func (c *Chain) QueryPacketCommitments(offset, limit, height uint64) (comRes *chantypes.QueryPacketCommitmentsResponse, err error) {
	return c.client.chanQuery.PacketCommitments(
		context.TODO(),
		&chantypes.QueryPacketCommitmentsRequest{
			PortId:     c.pathEnd.PortID,
			ChannelId:  c.pathEnd.ChannelID,
			Pagination: makePagination(offset, limit),
		},
	)
}

// QueryUnrecievedPackets returns a list of unrelayed packet commitments
func (c *Chain) QueryUnrecievedPackets(height uint64, seqs []uint64) ([]uint64, error) {
	res, err := c.client.chanQuery.UnreceivedPackets(
		context.TODO(),
		&chantypes.QueryUnreceivedPacketsRequest{
			PortId:                    c.pathEnd.PortID,
			ChannelId:                 c.pathEnd.ChannelID,
			PacketCommitmentSequences: seqs,
		},
	)
	if err != nil {
		return nil, err
	}
	return res.Sequences, nil
}

// QueryPacketAcknowledgements returns an array of packet acks
func (c *Chain) QueryPacketAcknowledgements(offset, limit, height uint64) (comRes *chantypes.QueryPacketAcknowledgementsResponse, err error) {
	return c.client.chanQuery.PacketAcknowledgements(
		context.TODO(),
		&chantypes.QueryPacketAcknowledgementsRequest{
			PortId:     c.pathEnd.PortID,
			ChannelId:  c.pathEnd.ChannelID,
			Pagination: makePagination(offset, limit),
		},
	)
}

// QueryUnrecievedAcknowledgements returns a list of unrelayed packet acks
func (c *Chain) QueryUnrecievedAcknowledgements(height uint64, seqs []uint64) ([]uint64, error) {
	res, err := c.client.chanQuery.UnreceivedAcks(
		context.TODO(),
		&chantypes.QueryUnreceivedAcksRequest{
			PortId:             c.pathEnd.PortID,
			ChannelId:          c.pathEnd.ChannelID,
			PacketAckSequences: seqs,
		},
	)
	if err != nil {
		return nil, err
	}
	return res.Sequences, nil
}

// QueryPacketAcknowledgementCommitment returns the packet ack proof at a given height
func (c *Chain) QueryPacketAcknowledgementCommitment(height int64, seq uint64) (ackRes *chantypes.QueryPacketAcknowledgementResponse, err error) {
	return c.client.chanQuery.PacketAcknowledgement(
		context.TODO(),
		&chantypes.QueryPacketAcknowledgementRequest{
			PortId:    c.pathEnd.PortID,
			ChannelId: c.pathEnd.ChannelID,
			Sequence:  seq,
		},
	)
}

// QueryPacket returns a packet corresponds to a given sequence
func (c *Chain) QueryPacket(height int64, sequence uint64) (*chantypes.Packet, error) {
	res, err := c.client.chanQuery.PacketCommitment(
		context.TODO(),
		&chantypes.QueryPacketCommitmentRequest{
			PortId:    c.pathEnd.PortID,
			ChannelId: c.pathEnd.ChannelID,
			Sequence:  sequence,
		},
	)
	if err != nil {
		return nil, err
	}

	// In Corda-IBC, packet commitment = marshaled packet :)
	var packet chantypes.Packet
	if err := packet.Unmarshal(res.Commitment); err != nil {
		return nil, err
	}

	return &packet, nil
}

func (c *Chain) QueryPacketAcknowledgement(height int64, sequence uint64) ([]byte, error) {
	res, err := c.client.chanQuery.PacketAcknowledgement(
		context.TODO(),
		&chantypes.QueryPacketAcknowledgementRequest{
			PortId:    c.pathEnd.PortID,
			ChannelId: c.pathEnd.ChannelID,
			Sequence:  sequence,
		},
	)
	if err != nil {
		return nil, err
	}
	// In Corda-IBC, ack commitment = marshaled ack
	return res.Acknowledgement, nil
}

func makePagination(offset, limit uint64) *query.PageRequest {
	return &query.PageRequest{
		Key:        []byte(""),
		Offset:     offset,
		Limit:      limit,
		CountTotal: true,
	}
}
