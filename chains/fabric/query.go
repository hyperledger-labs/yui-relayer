package fabric

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	querytypes "github.com/cosmos/cosmos-sdk/types/query"
	bankTypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	transfertypes "github.com/cosmos/ibc-go/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	conntypes "github.com/cosmos/ibc-go/modules/core/03-connection/types"
	chantypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
	committypes "github.com/cosmos/ibc-go/modules/core/23-commitment/types"
	ibcexported "github.com/cosmos/ibc-go/modules/core/exported"
	"github.com/gogo/protobuf/proto"
	"github.com/hyperledger-labs/yui-fabric-ibc/app"
	"github.com/hyperledger-labs/yui-fabric-ibc/chaincode"
	"github.com/hyperledger-labs/yui-fabric-ibc/commitment"
)

const (
	queryFunc                   = "query"
	getSequenceFunc             = "getSequence"
	queryPacketFunc             = "queryPacket"
	queryPacketAcknowledgement  = "queryPacketAcknowledgement"
	queryPacketAcknowledgements = "queryPacketAcknowledgements"
)

func (c *Chain) Query(req app.RequestQuery) (*app.ResponseQuery, error) {
	bz, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	r, err := c.Contract().EvaluateTransaction(queryFunc, string(bz))
	if err != nil {
		return nil, err
	}
	var res app.ResponseQuery
	if err := json.Unmarshal(r, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// QueryClientState retrevies the latest consensus state for a client in state at a given height
func (c *Chain) QueryClientState(_ int64) (*clienttypes.QueryClientStateResponse, error) {
	req := &clienttypes.QueryClientStateRequest{
		ClientId: c.pathEnd.ClientID,
	}
	var cres clienttypes.QueryClientStateResponse
	if err := c.query("/ibc.core.client.v1.Query/ClientState", req, &cres); err != nil {
		return nil, err
	}
	return &cres, nil
}

func (c *Chain) QueryClientConsensusState(_ int64, dstClientConsHeight ibcexported.Height) (*clienttypes.QueryConsensusStateResponse, error) {
	req := &clienttypes.QueryConsensusStateRequest{
		ClientId:       c.Path().ClientID,
		RevisionNumber: dstClientConsHeight.GetRevisionNumber(),
		RevisionHeight: dstClientConsHeight.GetRevisionHeight(),
	}
	var cres clienttypes.QueryConsensusStateResponse
	if err := c.query("/ibc.core.client.v1.Query/ConsensusState", req, &cres); err != nil {
		return nil, err
	}
	return &cres, nil
}

// QueryConnection returns the remote end of a given connection
func (c *Chain) QueryConnection(_ int64) (*conntypes.QueryConnectionResponse, error) {
	req := &conntypes.QueryConnectionRequest{
		ConnectionId: c.pathEnd.ConnectionID,
	}
	var cres conntypes.QueryConnectionResponse
	if err := c.query("/ibc.core.connection.v1.Query/Connection", req, &cres); err != nil {
		if strings.Contains(err.Error(), conntypes.ErrConnectionNotFound.Error()) {
			return emptyConnRes, nil
		}
		return nil, err
	}
	return &cres, nil
}

var emptyConnRes = conntypes.NewQueryConnectionResponse(
	conntypes.NewConnectionEnd(
		conntypes.UNINITIALIZED,
		"client",
		conntypes.NewCounterparty(
			"client",
			"connection",
			committypes.NewMerklePrefix([]byte{}),
		),
		[]*conntypes.Version{},
		0,
	),
	[]byte{},
	clienttypes.NewHeight(0, 0),
)

// QueryChannel returns the channel associated with a channelID
func (c *Chain) QueryChannel(height int64) (chanRes *chantypes.QueryChannelResponse, err error) {
	req := &chantypes.QueryChannelRequest{
		PortId:    c.pathEnd.PortID,
		ChannelId: c.pathEnd.ChannelID,
	}
	var cres chantypes.QueryChannelResponse
	if err := c.query("/ibc.core.channel.v1.Query/Channel", req, &cres); err != nil {
		if strings.Contains(err.Error(), chantypes.ErrChannelNotFound.Error()) {
			return emptyChannelRes, nil
		}
		return nil, err
	}
	return &cres, nil
}

func (c *Chain) QueryPacketCommitment(_ int64, seq uint64) (comRes *chantypes.QueryPacketCommitmentResponse, err error) {
	req := &chantypes.QueryPacketCommitmentRequest{
		PortId:    c.pathEnd.PortID,
		ChannelId: c.pathEnd.ChannelID,
		Sequence:  seq,
	}
	var res chantypes.QueryPacketCommitmentResponse
	if err := c.query("/ibc.core.channel.v1.Query/PacketCommitment", req, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

func (c *Chain) QueryPacketAcknowledgementCommitment(_ int64, seq uint64) (ackRes *chantypes.QueryPacketAcknowledgementResponse, err error) {
	req := &chantypes.QueryPacketAcknowledgementRequest{
		PortId:    c.pathEnd.PortID,
		ChannelId: c.pathEnd.ChannelID,
		Sequence:  seq,
	}
	var res chantypes.QueryPacketAcknowledgementResponse
	if err := c.query("/ibc.core.channel.v1.Query/PacketAcknowledgement", req, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// QueryCurrentSequence returns the current sequence for IBC chaincode
func (c *Chain) QueryCurrentSequence() (*commitment.Sequence, error) {
	res, err := c.Contract().EvaluateTransaction(getSequenceFunc)
	if err != nil {
		return nil, err
	}
	seq := new(commitment.Sequence)
	if err := json.Unmarshal(res, seq); err != nil {
		return nil, err
	}
	return seq, nil
}

// QueryBalance returns the amount of coins in the relayer account
func (c *Chain) QueryBalance(address sdk.AccAddress) (sdk.Coins, error) {
	req := bankTypes.NewQueryAllBalancesRequest(address, &querytypes.PageRequest{
		Key:        []byte(""),
		Offset:     0,
		Limit:      1000,
		CountTotal: true,
	})

	var res bankTypes.QueryAllBalancesResponse
	if err := c.query("/cosmos.bank.v1beta1.Query/AllBalances", req, &res); err != nil {
		return nil, err
	}
	return res.Balances, nil
}

// QueryDenomTraces returns all the denom traces from a given chain
func (c *Chain) QueryDenomTraces(offset, limit uint64, height int64) (*transfertypes.QueryDenomTracesResponse, error) {
	req := &transfertypes.QueryDenomTracesRequest{
		Pagination: &querytypes.PageRequest{
			Key:        []byte(""),
			Offset:     offset,
			Limit:      limit,
			CountTotal: true,
		},
	}
	var res transfertypes.QueryDenomTracesResponse
	if err := c.query("/ibc.applications.transfer.v1.Query/DenomTraces", req, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

func (c *Chain) QueryPacketCommitments(offset, limit uint64, height int64) (comRes *chantypes.QueryPacketCommitmentsResponse, err error) {
	req := &chantypes.QueryPacketCommitmentsRequest{
		PortId:    c.Path().PortID,
		ChannelId: c.Path().ChannelID,
		Pagination: &querytypes.PageRequest{
			Offset:     offset,
			Limit:      limit,
			CountTotal: true,
		},
	}
	var res chantypes.QueryPacketCommitmentsResponse
	if err := c.query("/ibc.core.channel.v1.Query/PacketCommitments", req, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

func (c *Chain) QueryUnrecievedPackets(height int64, seqs []uint64) ([]uint64, error) {
	req := &chantypes.QueryUnreceivedPacketsRequest{
		PortId:                    c.Path().PortID,
		ChannelId:                 c.Path().ChannelID,
		PacketCommitmentSequences: seqs,
	}
	var res chantypes.QueryUnreceivedPacketsResponse
	if err := c.query("/ibc.core.channel.v1.Query/UnreceivedPackets", req, &res); err != nil {
		return nil, err
	}
	return res.Sequences, nil
}

func (c *Chain) QueryPacketAcknowledgementCommitments(offset, limit uint64, _ int64) (comRes *chantypes.QueryPacketAcknowledgementsResponse, err error) {
	req := &chantypes.QueryPacketAcknowledgementsRequest{
		PortId:    c.Path().PortID,
		ChannelId: c.Path().ChannelID,
		Pagination: &querytypes.PageRequest{
			Offset:     offset,
			Limit:      limit,
			CountTotal: true,
		},
	}
	var res chantypes.QueryPacketAcknowledgementsResponse
	if err := c.query("/ibc.core.channel.v1.Query/PacketAcknowledgements", req, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

func (c *Chain) QueryUnrecievedAcknowledgements(_ int64, seqs []uint64) ([]uint64, error) {
	req := &chantypes.QueryUnreceivedAcksRequest{
		PortId:             c.Path().PortID,
		ChannelId:          c.Path().ChannelID,
		PacketAckSequences: seqs,
	}
	var res chantypes.QueryUnreceivedAcksResponse
	if err := c.query("/ibc.core.channel.v1.Query/UnreceivedAcks", req, &res); err != nil {
		return nil, err
	}
	return res.Sequences, nil
}

func (c *Chain) QueryPacket(_ int64, sequence uint64) (*chantypes.Packet, error) {
	var p chantypes.Packet

	bz, err := c.Contract().EvaluateTransaction(queryPacketFunc, c.Path().PortID, c.Path().ChannelID, fmt.Sprint(sequence))
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(bz, &p); err != nil {
		return nil, err
	}

	return &p, nil
}

func (dst *Chain) QueryPacketAcknowledgement(_ int64, sequence uint64) ([]byte, error) {
	bz, err := dst.Contract().EvaluateTransaction(queryPacketAcknowledgement, dst.Path().PortID, dst.Path().ChannelID, fmt.Sprint(sequence))
	if err != nil {
		return nil, err
	}
	return base64.StdEncoding.DecodeString(string(bz))
}

func (c *Chain) query(path string, req proto.Message, res interface{ Unmarshal(bz []byte) error }) error {
	bz, err := proto.Marshal(req)
	if err != nil {
		return err
	}
	r, err := c.Query(app.RequestQuery{
		Data: chaincode.EncodeToString(bz),
		Path: path,
	})
	if err != nil {
		return err
	}
	bz, err = chaincode.DecodeString(r.Value)
	if err != nil {
		return err
	}
	return res.Unmarshal(bz)
}
