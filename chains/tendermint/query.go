package tendermint

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	querytypes "github.com/cosmos/cosmos-sdk/types/query"
	bankTypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	transfertypes "github.com/cosmos/ibc-go/modules/apps/transfer/types"
	clientutils "github.com/cosmos/ibc-go/modules/core/02-client/client/utils"
	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	connutils "github.com/cosmos/ibc-go/modules/core/03-connection/client/utils"
	conntypes "github.com/cosmos/ibc-go/modules/core/03-connection/types"
	chanutils "github.com/cosmos/ibc-go/modules/core/04-channel/client/utils"
	chantypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
	committypes "github.com/cosmos/ibc-go/modules/core/23-commitment/types"
	ibcexported "github.com/cosmos/ibc-go/modules/core/exported"
	"github.com/hyperledger-labs/yui-relayer/core"
	abci "github.com/tendermint/tendermint/abci/types"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	tmtypes "github.com/tendermint/tendermint/types"
)

// QueryClientState retrevies the latest consensus state for a client in state at a given height
func (c *Chain) QueryClientState(height int64) (*clienttypes.QueryClientStateResponse, error) {
	return c.queryClientState(height, false)
}

func (c *Chain) queryClientState(height int64, _ bool) (*clienttypes.QueryClientStateResponse, error) {
	return clientutils.QueryClientStateABCI(c.CLIContext(height), c.PathEnd.ClientID)
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

// QueryConnection returns the remote end of a given connection
func (c *Chain) QueryConnection(height int64) (*conntypes.QueryConnectionResponse, error) {
	return c.queryConnection(height, false)
}

func (c *Chain) queryConnection(height int64, prove bool) (*conntypes.QueryConnectionResponse, error) {
	res, err := connutils.QueryConnection(c.CLIContext(height), c.PathEnd.ConnectionID, prove)
	if err != nil && strings.Contains(err.Error(), "not found") {
		return emptyConnRes, nil
	} else if err != nil {
		return nil, err
	}
	return res, nil
}

var emptyChannelRes = chantypes.NewQueryChannelResponse(
	chantypes.NewChannel(
		chantypes.UNINITIALIZED,
		chantypes.UNORDERED,
		chantypes.NewCounterparty(
			"port",
			"channel",
		),
		[]string{},
		"version",
	),
	[]byte{},
	clienttypes.NewHeight(0, 0),
)

// QueryChannel returns the channel associated with a channelID
func (c *Chain) QueryChannel(height int64) (chanRes *chantypes.QueryChannelResponse, err error) {
	return c.queryChannel(height, false)
}

func (c *Chain) queryChannel(height int64, prove bool) (chanRes *chantypes.QueryChannelResponse, err error) {
	res, err := chanutils.QueryChannel(c.CLIContext(height), c.PathEnd.PortID, c.PathEnd.ChannelID, prove)
	if err != nil && strings.Contains(err.Error(), "not found") {
		return emptyChannelRes, nil
	} else if err != nil {
		return nil, err
	}
	return res, nil
}

// QueryClientConsensusState retrevies the latest consensus state for a client in state at a given height
func (c *Chain) QueryClientConsensusState(
	height int64, dstClientConsHeight ibcexported.Height) (*clienttypes.QueryConsensusStateResponse, error) {
	return c.queryClientConsensusState(height, dstClientConsHeight, false)
}

func (c *Chain) queryClientConsensusState(
	height int64, dstClientConsHeight ibcexported.Height, _ bool) (*clienttypes.QueryConsensusStateResponse, error) {
	return clientutils.QueryConsensusStateABCI(
		c.CLIContext(height),
		c.PathEnd.ClientID,
		dstClientConsHeight,
	)
}

// QueryBalance returns the amount of coins in the relayer account
func (c *Chain) QueryBalance(addr sdk.AccAddress) (sdk.Coins, error) {
	params := bankTypes.NewQueryAllBalancesRequest(addr, &querytypes.PageRequest{
		Key:        []byte(""),
		Offset:     0,
		Limit:      1000,
		CountTotal: true,
	})

	queryClient := bankTypes.NewQueryClient(c.CLIContext(0))

	res, err := queryClient.AllBalances(context.Background(), params)
	if err != nil {
		return nil, err
	}

	return res.Balances, nil
}

// QueryDenomTraces returns all the denom traces from a given chain
func (c *Chain) QueryDenomTraces(offset, limit uint64, height int64) (*transfertypes.QueryDenomTracesResponse, error) {
	return transfertypes.NewQueryClient(c.CLIContext(height)).DenomTraces(context.Background(), &transfertypes.QueryDenomTracesRequest{
		Pagination: &querytypes.PageRequest{
			Key:        []byte(""),
			Offset:     offset,
			Limit:      limit,
			CountTotal: true,
		},
	})
}

// QueryPacketCommitment returns the packet commitment proof at a given height
func (c *Chain) QueryPacketCommitment(
	height int64, seq uint64) (comRes *chantypes.QueryPacketCommitmentResponse, err error) {
	return c.queryPacketCommitment(height, seq, false)
}

func (c *Chain) queryPacketCommitment(
	height int64, seq uint64, prove bool) (comRes *chantypes.QueryPacketCommitmentResponse, err error) {
	return chanutils.QueryPacketCommitment(c.CLIContext(height), c.PathEnd.PortID, c.PathEnd.ChannelID, seq, prove)
}

// QueryPacketAcknowledgementCommitment returns the packet ack proof at a given height
func (c *Chain) QueryPacketAcknowledgementCommitment(height int64, seq uint64) (ackRes *chantypes.QueryPacketAcknowledgementResponse, err error) {
	return c.queryPacketAcknowledgementCommitment(height, seq, false)
}

func (c *Chain) queryPacketAcknowledgementCommitment(height int64, seq uint64, prove bool) (ackRes *chantypes.QueryPacketAcknowledgementResponse, err error) {
	return chanutils.QueryPacketAcknowledgement(c.CLIContext(height), c.PathEnd.PortID, c.PathEnd.ChannelID, seq, prove)
}

func (dst *Chain) QueryPacketAcknowledgement(height int64, sequence uint64) ([]byte, error) {
	txs, err := dst.QueryTxs(height, 1, 1000, ackPacketQuery(dst.Path().ChannelID, int(sequence)))
	switch {
	case err != nil:
		return nil, err
	case len(txs) == 0:
		return nil, fmt.Errorf("no transactions returned with query")
	case len(txs) > 1:
		return nil, fmt.Errorf("more than one transaction returned with query")
	}

	ack, err := core.FindPacketAcknowledgementFromEventsBySequence(txs[0].TxResult.Events, sequence)
	if err != nil {
		return nil, err
	}
	if ack == nil {
		return nil, fmt.Errorf("can't find the packet from events")
	}
	return ack.Data(), nil
}

// QueryPacketReciept returns the packet reciept proof at a given height
func (c *Chain) QueryPacketReciept(height int64, seq uint64) (recRes *chantypes.QueryPacketReceiptResponse, err error) {
	return chanutils.QueryPacketReceipt(c.CLIContext(height), c.PathEnd.PortID, c.PathEnd.ChannelID, seq, true)
}

// QueryPacketCommitments returns an array of packet commitments
func (c *Chain) QueryPacketCommitments(
	offset, limit uint64, height int64) (comRes *chantypes.QueryPacketCommitmentsResponse, err error) {
	qc := chantypes.NewQueryClient(c.CLIContext(int64(height)))
	return qc.PacketCommitments(context.Background(), &chantypes.QueryPacketCommitmentsRequest{
		PortId:    c.PathEnd.PortID,
		ChannelId: c.PathEnd.ChannelID,
		Pagination: &querytypes.PageRequest{
			Offset:     offset,
			Limit:      limit,
			CountTotal: true,
		},
	})
}

// QueryPacketAcknowledgementCommitments returns an array of packet acks
func (c *Chain) QueryPacketAcknowledgementCommitments(offset, limit uint64, height int64) (comRes *chantypes.QueryPacketAcknowledgementsResponse, err error) {
	qc := chantypes.NewQueryClient(c.CLIContext(int64(height)))
	return qc.PacketAcknowledgements(context.Background(), &chantypes.QueryPacketAcknowledgementsRequest{
		PortId:    c.PathEnd.PortID,
		ChannelId: c.PathEnd.ChannelID,
		Pagination: &querytypes.PageRequest{
			Offset:     offset,
			Limit:      limit,
			CountTotal: true,
		},
	})
}

// QueryUnrecievedPackets returns a list of unrelayed packet commitments
func (c *Chain) QueryUnrecievedPackets(height int64, seqs []uint64) ([]uint64, error) {
	qc := chantypes.NewQueryClient(c.CLIContext(int64(height)))
	res, err := qc.UnreceivedPackets(context.Background(), &chantypes.QueryUnreceivedPacketsRequest{
		PortId:                    c.PathEnd.PortID,
		ChannelId:                 c.PathEnd.ChannelID,
		PacketCommitmentSequences: seqs,
	})
	if err != nil {
		return nil, err
	}
	return res.Sequences, nil
}

// QueryUnrecievedAcknowledgements returns a list of unrelayed packet acks
func (c *Chain) QueryUnrecievedAcknowledgements(height int64, seqs []uint64) ([]uint64, error) {
	qc := chantypes.NewQueryClient(c.CLIContext(int64(height)))
	res, err := qc.UnreceivedAcks(context.Background(), &chantypes.QueryUnreceivedAcksRequest{
		PortId:             c.PathEnd.PortID,
		ChannelId:          c.PathEnd.ChannelID,
		PacketAckSequences: seqs,
	})
	if err != nil {
		return nil, err
	}
	return res.Sequences, nil
}

func (src *Chain) QueryPacket(height int64, seq uint64) (*chantypes.Packet, error) {
	txs, err := src.QueryTxs(height, 1, 1000, rcvPacketQuery(src.Path().ChannelID, int(seq)))
	switch {
	case err != nil:
		return nil, err
	case len(txs) == 0:
		return nil, fmt.Errorf("no transactions returned with query")
	case len(txs) > 1:
		return nil, fmt.Errorf("more than one transaction returned with query")
	}

	packet, err := core.FindPacketFromEventsBySequence(txs[0].TxResult.Events, seq)
	if err != nil {
		return nil, err
	}
	if packet == nil {
		return nil, fmt.Errorf("can't find the packet from events")
	}
	return packet, nil
}

// QueryTxs returns an array of transactions given a tag
func (c *Chain) QueryTxs(height int64, page, limit int, events []string) ([]*ctypes.ResultTx, error) {
	if len(events) == 0 {
		return nil, errors.New("must declare at least one event to search")
	}

	if page <= 0 {
		return nil, errors.New("page must greater than 0")
	}

	if limit <= 0 {
		return nil, errors.New("limit must greater than 0")
	}

	res, err := c.Client.TxSearch(context.Background(), strings.Join(events, " AND "), true, &page, &limit, "")
	if err != nil {
		return nil, err
	}
	return res.Txs, nil
}

/////////////////////////////////////
//    STAKING -> HistoricalInfo     //
/////////////////////////////////////

// QueryHistoricalInfo returns historical header data
func (c *Chain) QueryHistoricalInfo(height clienttypes.Height) (*stakingtypes.QueryHistoricalInfoResponse, error) {
	//TODO: use epoch number in query once SDK gets updated
	qc := stakingtypes.NewQueryClient(c.CLIContext(int64(height.GetRevisionHeight())))
	return qc.HistoricalInfo(context.Background(), &stakingtypes.QueryHistoricalInfoRequest{
		Height: int64(height.GetRevisionHeight()),
	})
}

// QueryValsetAtHeight returns the validator set at a given height
func (c *Chain) QueryValsetAtHeight(height clienttypes.Height) (*tmproto.ValidatorSet, error) {
	res, err := c.QueryHistoricalInfo(height)
	if err != nil {
		return nil, err
	}

	// create tendermint ValidatorSet from SDK Validators
	tmVals, err := c.toTmValidators(res.Hist.Valset)
	if err != nil {
		return nil, err
	}

	sort.Sort(tmtypes.ValidatorsByVotingPower(tmVals))
	tmValSet := &tmtypes.ValidatorSet{
		Validators: tmVals,
	}
	tmValSet.GetProposer()

	return tmValSet.ToProto()
}

func (c *Chain) toTmValidators(vals stakingtypes.Validators) ([]*tmtypes.Validator, error) {
	validators := make([]*tmtypes.Validator, len(vals))
	var err error
	for i, val := range vals {
		validators[i], err = c.toTmValidator(val)
		if err != nil {
			return nil, err
		}
	}

	return validators, nil
}

func (c *Chain) toTmValidator(val stakingtypes.Validator) (*tmtypes.Validator, error) {
	var pk cryptotypes.PubKey
	if err := c.codec.UnpackAny(val.ConsensusPubkey, &pk); err != nil {
		return nil, err
	}
	tmkey, err := cryptocodec.ToTmPubKeyInterface(pk)
	if err != nil {
		return nil, fmt.Errorf("pubkey not a tendermint pub key %s", err)
	}
	return tmtypes.NewValidator(tmkey, val.ConsensusPower(sdk.DefaultPowerReduction)), nil
}

// QueryUnbondingPeriod returns the unbonding period of the chain
func (c *Chain) QueryUnbondingPeriod() (time.Duration, error) {
	req := stakingtypes.QueryParamsRequest{}

	queryClient := stakingtypes.NewQueryClient(c.CLIContext(0))

	res, err := queryClient.Params(context.Background(), &req)
	if err != nil {
		return 0, err
	}

	return res.Params.UnbondingTime, nil
}

// QueryConsensusParams returns the consensus params
func (c *Chain) QueryConsensusParams() (*abci.ConsensusParams, error) {
	rg, err := c.Client.Genesis(context.Background())
	if err != nil {
		return nil, err
	}

	return tmtypes.TM2PB.ConsensusParams(rg.Genesis.ConsensusParams), nil
}

const (
	spTag = "send_packet"
	waTag = "write_acknowledgement"
)

func rcvPacketQuery(channelID string, seq int) []string {
	return []string{fmt.Sprintf("%s.packet_src_channel='%s'", spTag, channelID), fmt.Sprintf("%s.packet_sequence='%d'", spTag, seq)}
}

func ackPacketQuery(channelID string, seq int) []string {
	return []string{fmt.Sprintf("%s.packet_dst_channel='%s'", waTag, channelID), fmt.Sprintf("%s.packet_sequence='%d'", waTag, seq)}
}
