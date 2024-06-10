package tendermint

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	ctypes "github.com/cometbft/cometbft/rpc/core/types"
	tmtypes "github.com/cometbft/cometbft/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	querytypes "github.com/cosmos/cosmos-sdk/types/query"
	bankTypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	clientutils "github.com/cosmos/ibc-go/v8/modules/core/02-client/client/utils"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	connutils "github.com/cosmos/ibc-go/v8/modules/core/03-connection/client/utils"
	conntypes "github.com/cosmos/ibc-go/v8/modules/core/03-connection/types"
	chanutils "github.com/cosmos/ibc-go/v8/modules/core/04-channel/client/utils"
	chantypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	committypes "github.com/cosmos/ibc-go/v8/modules/core/23-commitment/types"
	ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"
	"github.com/hyperledger-labs/yui-relayer/core"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// QueryClientState retrevies the latest consensus state for a client in state at a given height
func (c *Chain) QueryClientState(ctx core.QueryContext) (*clienttypes.QueryClientStateResponse, error) {
	return c.queryClientState(int64(ctx.Height().GetRevisionHeight()), false)
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
func (c *Chain) QueryConnection(ctx core.QueryContext, connectionID string) (*conntypes.QueryConnectionResponse, error) {
	return c.queryConnection(int64(ctx.Height().GetRevisionHeight()), connectionID, false)
}

func (c *Chain) queryConnection(height int64, connectionID string, prove bool) (*conntypes.QueryConnectionResponse, error) {
	res, err := connutils.QueryConnection(c.CLIContext(height), connectionID, prove)
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
func (c *Chain) QueryChannel(ctx core.QueryContext) (chanRes *chantypes.QueryChannelResponse, err error) {
	return c.queryChannel(int64(ctx.Height().GetRevisionHeight()), false)
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
	ctx core.QueryContext, dstClientConsHeight ibcexported.Height) (*clienttypes.QueryConsensusStateResponse, error) {
	return c.queryClientConsensusState(int64(ctx.Height().GetRevisionHeight()), dstClientConsHeight, false)
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
func (c *Chain) QueryBalance(ctx core.QueryContext, addr sdk.AccAddress) (sdk.Coins, error) {
	params := bankTypes.NewQueryAllBalancesRequest(addr, &querytypes.PageRequest{
		Key:        []byte(""),
		Offset:     0,
		Limit:      1000,
		CountTotal: true,
	}, true)

	queryClient := bankTypes.NewQueryClient(c.CLIContext(0))

	res, err := queryClient.AllBalances(context.Background(), params)
	if err != nil {
		return nil, err
	}

	return res.Balances, nil
}

// QueryDenomTraces returns all the denom traces from a given chain
func (c *Chain) QueryDenomTraces(ctx core.QueryContext, offset, limit uint64) (*transfertypes.QueryDenomTracesResponse, error) {
	return transfertypes.NewQueryClient(c.CLIContext(int64(ctx.Height().GetRevisionHeight()))).DenomTraces(context.Background(), &transfertypes.QueryDenomTracesRequest{
		Pagination: &querytypes.PageRequest{
			Key:        []byte(""),
			Offset:     offset,
			Limit:      limit,
			CountTotal: true,
		},
	})
}

// queryPacketCommitments returns an array of packet commitments
func (c *Chain) queryPacketCommitments(
	ctx core.QueryContext,
	offset, limit uint64) (comRes *chantypes.QueryPacketCommitmentsResponse, err error) {
	qc := chantypes.NewQueryClient(c.CLIContext(int64(ctx.Height().GetRevisionHeight())))
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

// queryPacketAcknowledgementCommitments returns an array of packet acks
func (c *Chain) queryPacketAcknowledgementCommitments(ctx core.QueryContext, offset, limit uint64) (comRes *chantypes.QueryPacketAcknowledgementsResponse, err error) {
	qc := chantypes.NewQueryClient(c.CLIContext(int64(ctx.Height().GetRevisionHeight())))
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

// QueryUnreceivedPackets returns a list of unrelayed packet commitments
func (c *Chain) QueryUnreceivedPackets(ctx core.QueryContext, seqs []uint64) ([]uint64, error) {
	qc := chantypes.NewQueryClient(c.CLIContext(int64(ctx.Height().GetRevisionHeight())))
	res, err := qc.UnreceivedPackets(context.Background(), &chantypes.QueryUnreceivedPacketsRequest{
		PortId:                    c.PathEnd.PortID,
		ChannelId:                 c.PathEnd.ChannelID,
		PacketCommitmentSequences: seqs,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query unreceived packets: error=%w height=%v", err, ctx.Height())
	}
	return res.Sequences, nil
}

func (c *Chain) QueryUnfinalizedRelayPackets(ctx core.QueryContext, counterparty core.LightClientICS04Querier) (core.PacketInfoList, error) {
	res, err := c.queryPacketCommitments(ctx, 0, 1000)
	if err != nil {
		return nil, fmt.Errorf("failed to query packet commitments: error=%w height=%v", err, ctx.Height())
	}

	var packets core.PacketInfoList
	for _, ps := range res.Commitments {
		packet, height, err := c.querySentPacket(ctx, ps.Sequence)
		if err != nil {
			return nil, fmt.Errorf("failed to query sent packet: error=%w height=%v", err, ctx.Height())
		}
		packets = append(packets, &core.PacketInfo{
			Packet:          *packet,
			Acknowledgement: nil,
			EventHeight:     height,
		})
	}

	var counterpartyCtx core.QueryContext
	if counterpartyH, err := counterparty.GetLatestFinalizedHeader(); err != nil {
		return nil, fmt.Errorf("failed to get latest finalized header: error=%w height=%v", err, ctx.Height())
	} else {
		counterpartyCtx = core.NewQueryContext(context.TODO(), counterpartyH.GetHeight())
	}

	seqs, err := counterparty.QueryUnreceivedPackets(counterpartyCtx, packets.ExtractSequenceList())
	if err != nil {
		return nil, fmt.Errorf("failed to query counterparty for unreceived packets: error=%w, height=%v", err, counterpartyCtx.Height())
	}
	packets = packets.Filter(seqs)

	return packets, nil
}

// QueryUnreceivedAcknowledgements returns a list of unrelayed packet acks
func (c *Chain) QueryUnreceivedAcknowledgements(ctx core.QueryContext, seqs []uint64) ([]uint64, error) {
	qc := chantypes.NewQueryClient(c.CLIContext(int64(ctx.Height().GetRevisionHeight())))
	res, err := qc.UnreceivedAcks(context.Background(), &chantypes.QueryUnreceivedAcksRequest{
		PortId:             c.PathEnd.PortID,
		ChannelId:          c.PathEnd.ChannelID,
		PacketAckSequences: seqs,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query unreceived acks: : error=%w height=%v", err, ctx.Height())
	}
	return res.Sequences, nil
}

func (c *Chain) QueryUnfinalizedRelayAcknowledgements(ctx core.QueryContext, counterparty core.LightClientICS04Querier) (core.PacketInfoList, error) {
	res, err := c.queryPacketAcknowledgementCommitments(ctx, 0, 1000)
	if err != nil {
		return nil, fmt.Errorf("failed to query packet acknowledgement commitments: error=%w height=%v", err, ctx.Height())
	}

	var packets core.PacketInfoList
	for _, ps := range res.Acknowledgements {
		packet, rpHeight, err := c.queryReceivedPacket(ctx, ps.Sequence)
		if err != nil {
			return nil, fmt.Errorf("failed to query received packet: error=%w height=%v", err, ctx.Height())
		}
		ack, _, err := c.queryWrittenAcknowledgement(ctx, ps.Sequence)
		if err != nil {
			return nil, fmt.Errorf("failed to query written acknowledgement: error=%w height=%v", err, ctx.Height())
		}
		packets = append(packets, &core.PacketInfo{
			Packet:          *packet,
			Acknowledgement: ack,
			EventHeight:     rpHeight,
		})
	}

	var counterpartyCtx core.QueryContext
	if counterpartyH, err := counterparty.GetLatestFinalizedHeader(); err != nil {
		return nil, fmt.Errorf("failed to get latest finalized header: error=%w height=%v", err, ctx.Height())
	} else {
		counterpartyCtx = core.NewQueryContext(context.TODO(), counterpartyH.GetHeight())
	}

	seqs, err := counterparty.QueryUnreceivedAcknowledgements(counterpartyCtx, packets.ExtractSequenceList())
	if err != nil {
		return nil, fmt.Errorf("failed to query counterparty for unreceived acknowledgements: error=%w height=%v", err, counterpartyCtx.Height())
	}
	packets = packets.Filter(seqs)

	return packets, nil
}

// querySentPacket finds a SendPacket event corresponding to `seq` and returns the packet in it
func (c *Chain) querySentPacket(ctx core.QueryContext, seq uint64) (*chantypes.Packet, clienttypes.Height, error) {
	txs, err := c.QueryTxs(int64(ctx.Height().GetRevisionHeight()), 1, 1000, sendPacketQuery(c.Path().ChannelID, int(seq)))
	switch {
	case err != nil:
		return nil, clienttypes.Height{}, err
	case len(txs) == 0:
		return nil, clienttypes.Height{}, fmt.Errorf("no transactions returned with query")
	case len(txs) > 1:
		return nil, clienttypes.Height{}, fmt.Errorf("more than one transaction returned with query")
	}

	packet, err := core.FindPacketFromEventsBySequence(txs[0].TxResult.Events, chantypes.EventTypeSendPacket, seq)
	if err != nil {
		return nil, clienttypes.Height{}, err
	}
	if packet == nil {
		return nil, clienttypes.Height{}, fmt.Errorf("can't find the packet from events")
	}

	height := clienttypes.NewHeight(clienttypes.ParseChainID(c.ChainID()), uint64(txs[0].Height))

	return packet, height, nil
}

// queryReceivedPacket finds a RecvPacket event corresponding to `seq` and return the packet in it
func (c *Chain) queryReceivedPacket(ctx core.QueryContext, seq uint64) (*chantypes.Packet, clienttypes.Height, error) {
	txs, err := c.QueryTxs(int64(ctx.Height().GetRevisionHeight()), 1, 1000, recvPacketQuery(c.Path().ChannelID, int(seq)))
	switch {
	case err != nil:
		return nil, clienttypes.Height{}, err
	case len(txs) == 0:
		return nil, clienttypes.Height{}, fmt.Errorf("no transactions returned with query")
	case len(txs) > 1:
		return nil, clienttypes.Height{}, fmt.Errorf("more than one transaction returned with query")
	}

	packet, err := core.FindPacketFromEventsBySequence(txs[0].TxResult.Events, chantypes.EventTypeRecvPacket, seq)
	if err != nil {
		return nil, clienttypes.Height{}, err
	}
	if packet == nil {
		return nil, clienttypes.Height{}, fmt.Errorf("can't find the packet from events")
	}

	height := clienttypes.NewHeight(clienttypes.ParseChainID(c.ChainID()), uint64(txs[0].Height))

	return packet, height, nil

}

// queryWrittenAcknowledgement finds a WriteAcknowledgement event corresponding to `seq` and returns the acknowledgement in it
func (c *Chain) queryWrittenAcknowledgement(ctx core.QueryContext, seq uint64) ([]byte, clienttypes.Height, error) {
	txs, err := c.QueryTxs(int64(ctx.Height().GetRevisionHeight()), 1, 1000, writeAckQuery(c.Path().ChannelID, int(seq)))
	switch {
	case err != nil:
		return nil, clienttypes.Height{}, err
	case len(txs) == 0:
		return nil, clienttypes.Height{}, fmt.Errorf("no transactions returned with query")
	case len(txs) > 1:
		return nil, clienttypes.Height{}, fmt.Errorf("more than one transaction returned with query")
	}

	ack, err := core.FindPacketAcknowledgementFromEventsBySequence(txs[0].TxResult.Events, seq)
	if err != nil {
		return nil, clienttypes.Height{}, err
	}
	if ack == nil {
		return nil, clienttypes.Height{}, fmt.Errorf("can't find the packet from events")
	}

	height := clienttypes.NewHeight(clienttypes.ParseChainID(c.ChainID()), uint64(txs[0].Height))

	return ack.Data(), height, nil
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

func (c *Chain) QueryChannelUpgrade(ctx core.QueryContext) (*chantypes.QueryUpgradeResponse, error) {
	return c.queryChannelUpgrade(int64(ctx.Height().GetRevisionHeight()), false)
}

func (c *Chain) queryChannelUpgrade(height int64, prove bool) (chanRes *chantypes.QueryUpgradeResponse, err error) {
	if res, err := chanutils.QueryUpgrade(
		c.CLIContext(height),
		c.PathEnd.PortID,
		c.PathEnd.ChannelID,
		prove,
	); err != nil {
		if st, ok := status.FromError(err); ok && st.Code() == codes.NotFound {
			return nil, nil
		} else {
			return nil, err
		}
	} else {
		return res, nil
	}
}

func (c *Chain) QueryChannelUpgradeError(ctx core.QueryContext) (*chantypes.QueryUpgradeErrorResponse, error) {
	return c.queryChannelUpgradeError(int64(ctx.Height().GetRevisionHeight()), false)
}

func (c *Chain) queryChannelUpgradeError(height int64, prove bool) (chanRes *chantypes.QueryUpgradeErrorResponse, err error) {
	if res, err := chanutils.QueryUpgradeError(
		c.CLIContext(height),
		c.PathEnd.PortID,
		c.PathEnd.ChannelID,
		prove,
	); err != nil {
		if st, ok := status.FromError(err); ok && st.Code() == codes.NotFound {
			return nil, nil
		} else {
			return nil, err
		}
	} else {
		return res, nil
	}
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

	protoValSet, err := tmValSet.ToProto()
	if err != nil {
		return nil, err
	}
	protoValSet.TotalVotingPower = tmValSet.TotalVotingPower()
	return protoValSet, err
}

func (c *Chain) toTmValidators(vals []stakingtypes.Validator) ([]*tmtypes.Validator, error) {
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

const (
	spTag = "send_packet"
	rpTag = "recv_packet"
	waTag = "write_acknowledgement"
)

func sendPacketQuery(channelID string, seq int) []string {
	return []string{fmt.Sprintf("%s.packet_src_channel='%s'", spTag, channelID), fmt.Sprintf("%s.packet_sequence='%d'", spTag, seq)}
}

func recvPacketQuery(channelID string, seq int) []string {
	return []string{fmt.Sprintf("%s.packet_src_channel='%s'", rpTag, channelID), fmt.Sprintf("%s.packet_sequence='%d'", rpTag, seq)}
}

func writeAckQuery(channelID string, seq int) []string {
	return []string{fmt.Sprintf("%s.packet_dst_channel='%s'", waTag, channelID), fmt.Sprintf("%s.packet_sequence='%d'", waTag, seq)}
}
