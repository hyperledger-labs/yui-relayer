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
	otelcodes "go.opentelemetry.io/otel/codes"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// QueryClientState retrevies the latest consensus state for a client in state at a given height
func (c *Chain) QueryClientState(ctx core.QueryContext) (*clienttypes.QueryClientStateResponse, error) {
	return c.queryClientState(ctx.Context(), int64(ctx.Height().GetRevisionHeight()), false)
}

func (c *Chain) queryClientState(ctx context.Context, height int64, _ bool) (*clienttypes.QueryClientStateResponse, error) {
	return clientutils.QueryClientStateABCI(c.CLIContext(height).WithCmdContext(ctx), c.PathEnd.ClientID)
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
	return c.queryConnection(ctx.Context(), int64(ctx.Height().GetRevisionHeight()), connectionID, false)
}

func (c *Chain) queryConnection(ctx context.Context, height int64, connectionID string, prove bool) (*conntypes.QueryConnectionResponse, error) {
	res, err := connutils.QueryConnection(c.CLIContext(height).WithCmdContext(ctx), connectionID, prove)
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
	return c.queryChannel(ctx.Context(), int64(ctx.Height().GetRevisionHeight()), false)
}

func (c *Chain) queryChannel(ctx context.Context, height int64, prove bool) (chanRes *chantypes.QueryChannelResponse, err error) {
	res, err := chanutils.QueryChannel(c.CLIContext(height).WithCmdContext(ctx), c.PathEnd.PortID, c.PathEnd.ChannelID, prove)
	if err != nil && strings.Contains(err.Error(), "not found") {
		return emptyChannelRes, nil
	} else if err != nil {
		return nil, err
	}
	return res, nil
}

// QueryClientConsensusState retrieves the latest consensus state for a client in state at a given height
func (c *Chain) QueryClientConsensusState(
	ctx core.QueryContext, dstClientConsHeight ibcexported.Height) (*clienttypes.QueryConsensusStateResponse, error) {
	return c.queryClientConsensusState(ctx.Context(), int64(ctx.Height().GetRevisionHeight()), dstClientConsHeight, false)
}

func (c *Chain) queryClientConsensusState(
	ctx context.Context, height int64, dstClientConsHeight ibcexported.Height, _ bool) (*clienttypes.QueryConsensusStateResponse, error) {
	return clientutils.QueryConsensusStateABCI(
		c.CLIContext(height).WithCmdContext(ctx),
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

	queryClient := bankTypes.NewQueryClient(c.CLIContext(0).WithCmdContext(ctx.Context()))

	res, err := queryClient.AllBalances(ctx.Context(), params)
	if err != nil {
		return nil, err
	}

	return res.Balances, nil
}

// QueryDenomTraces returns all the denom traces from a given chain
func (c *Chain) QueryDenomTraces(ctx core.QueryContext, offset, limit uint64) (*transfertypes.QueryDenomTracesResponse, error) {
	return transfertypes.NewQueryClient(c.CLIContext(int64(ctx.Height().GetRevisionHeight())).WithCmdContext(ctx.Context())).DenomTraces(ctx.Context(), &transfertypes.QueryDenomTracesRequest{
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
	ctx, span := core.StartTraceWithQueryContext(tracer, ctx, "Chain.queryPacketCommitments", core.WithChainAttributes(c.ChainID()))
	defer span.End()

	qc := chantypes.NewQueryClient(c.CLIContext(int64(ctx.Height().GetRevisionHeight())).WithCmdContext(ctx.Context()))
	resp, err := qc.PacketCommitments(ctx.Context(), &chantypes.QueryPacketCommitmentsRequest{
		PortId:    c.PathEnd.PortID,
		ChannelId: c.PathEnd.ChannelID,
		Pagination: &querytypes.PageRequest{
			Offset:     offset,
			Limit:      limit,
			CountTotal: true,
		},
	})
	if err != nil {
		span.SetStatus(otelcodes.Error, err.Error())
		return nil, err
	}
	return resp, nil
}

// queryPacketAcknowledgementCommitments returns an array of packet acks
func (c *Chain) queryPacketAcknowledgementCommitments(ctx core.QueryContext, offset, limit uint64) (comRes *chantypes.QueryPacketAcknowledgementsResponse, err error) {
	ctx, span := core.StartTraceWithQueryContext(tracer, ctx, "Chain.queryPacketAcknowledgementCommitments", core.WithChainAttributes(c.ChainID()))
	defer span.End()

	qc := chantypes.NewQueryClient(c.CLIContext(int64(ctx.Height().GetRevisionHeight())).WithCmdContext(ctx.Context()))
	resp, err := qc.PacketAcknowledgements(ctx.Context(), &chantypes.QueryPacketAcknowledgementsRequest{
		PortId:    c.PathEnd.PortID,
		ChannelId: c.PathEnd.ChannelID,
		Pagination: &querytypes.PageRequest{
			Offset:     offset,
			Limit:      limit,
			CountTotal: true,
		},
	})
	if err != nil {
		span.SetStatus(otelcodes.Error, err.Error())
		return nil, err
	}
	return resp, nil
}

// QueryUnreceivedPackets returns a list of unrelayed packet commitments
func (c *Chain) QueryUnreceivedPackets(ctx core.QueryContext, seqs []uint64) ([]uint64, error) {
	qc := chantypes.NewQueryClient(c.CLIContext(int64(ctx.Height().GetRevisionHeight())).WithCmdContext(ctx.Context()))
	res, err := qc.UnreceivedPackets(ctx.Context(), &chantypes.QueryUnreceivedPacketsRequest{
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
	if counterpartyH, err := counterparty.GetLatestFinalizedHeader(ctx.Context()); err != nil {
		return nil, fmt.Errorf("failed to get latest finalized header: error=%w height=%v", err, ctx.Height())
	} else {
		counterpartyCtx = core.NewQueryContext(ctx.Context(), counterpartyH.GetHeight())
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
	qc := chantypes.NewQueryClient(c.CLIContext(int64(ctx.Height().GetRevisionHeight())).WithCmdContext(ctx.Context()))
	res, err := qc.UnreceivedAcks(ctx.Context(), &chantypes.QueryUnreceivedAcksRequest{
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
	if counterpartyH, err := counterparty.GetLatestFinalizedHeader(ctx.Context()); err != nil {
		return nil, fmt.Errorf("failed to get latest finalized header: error=%w height=%v", err, ctx.Height())
	} else {
		counterpartyCtx = core.NewQueryContext(ctx.Context(), counterpartyH.GetHeight())
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
	ctx, span := core.StartTraceWithQueryContext(tracer, ctx, "Chain.querySentPacket", core.WithChainAttributes(c.ChainID()))
	defer span.End()

	txs, err := c.QueryTxs(ctx.Context(), int64(ctx.Height().GetRevisionHeight()), 1, 1000, sendPacketQuery(c.Path().ChannelID, int(seq)))
	switch {
	case err != nil:
		span.SetStatus(otelcodes.Error, err.Error())
		return nil, clienttypes.Height{}, err
	case len(txs) == 0:
		err = fmt.Errorf("no transactions returned with query")
		span.SetStatus(otelcodes.Error, err.Error())
		return nil, clienttypes.Height{}, err
	case len(txs) > 1:
		err = fmt.Errorf("more than one transaction returned with query")
		span.SetStatus(otelcodes.Error, err.Error())
		return nil, clienttypes.Height{}, err
	}

	packet, err := core.FindPacketFromEventsBySequence(txs[0].TxResult.Events, chantypes.EventTypeSendPacket, seq)
	if err != nil {
		span.SetStatus(otelcodes.Error, err.Error())
		return nil, clienttypes.Height{}, err
	}
	if packet == nil {
		err = fmt.Errorf("can't find the packet from events")
		span.SetStatus(otelcodes.Error, err.Error())
		return nil, clienttypes.Height{}, err
	}

	height := clienttypes.NewHeight(clienttypes.ParseChainID(c.ChainID()), uint64(txs[0].Height))

	return packet, height, nil
}

// queryReceivedPacket finds a RecvPacket event corresponding to `seq` and return the packet in it
func (c *Chain) queryReceivedPacket(ctx core.QueryContext, seq uint64) (*chantypes.Packet, clienttypes.Height, error) {
	ctx, span := core.StartTraceWithQueryContext(tracer, ctx, "Chain.queryReceivedPacket", core.WithChainAttributes(c.ChainID()))
	defer span.End()

	txs, err := c.QueryTxs(ctx.Context(), int64(ctx.Height().GetRevisionHeight()), 1, 1000, recvPacketQuery(c.Path().ChannelID, int(seq)))
	switch {
	case err != nil:
		span.SetStatus(otelcodes.Error, err.Error())
		return nil, clienttypes.Height{}, err
	case len(txs) == 0:
		err = fmt.Errorf("no transactions returned with query")
		span.SetStatus(otelcodes.Error, err.Error())
		return nil, clienttypes.Height{}, err
	case len(txs) > 1:
		err = fmt.Errorf("more than one transaction returned with query")
		span.SetStatus(otelcodes.Error, err.Error())
		return nil, clienttypes.Height{}, err
	}

	packet, err := core.FindPacketFromEventsBySequence(txs[0].TxResult.Events, chantypes.EventTypeRecvPacket, seq)
	if err != nil {
		span.SetStatus(otelcodes.Error, err.Error())
		return nil, clienttypes.Height{}, err
	}
	if packet == nil {
		err = fmt.Errorf("can't find the packet from events")
		span.SetStatus(otelcodes.Error, err.Error())
		return nil, clienttypes.Height{}, err
	}

	height := clienttypes.NewHeight(clienttypes.ParseChainID(c.ChainID()), uint64(txs[0].Height))

	return packet, height, nil

}

// queryWrittenAcknowledgement finds a WriteAcknowledgement event corresponding to `seq` and returns the acknowledgement in it
func (c *Chain) queryWrittenAcknowledgement(ctx core.QueryContext, seq uint64) ([]byte, clienttypes.Height, error) {
	ctx, span := core.StartTraceWithQueryContext(tracer, ctx, "Chain.queryWrittenAcknowledgement", core.WithChainAttributes(c.ChainID()))
	defer span.End()

	txs, err := c.QueryTxs(ctx.Context(), int64(ctx.Height().GetRevisionHeight()), 1, 1000, writeAckQuery(c.Path().ChannelID, int(seq)))
	switch {
	case err != nil:
		span.SetStatus(otelcodes.Error, err.Error())
		return nil, clienttypes.Height{}, err
	case len(txs) == 0:
		err = fmt.Errorf("no transactions returned with query")
		span.SetStatus(otelcodes.Error, err.Error())
		return nil, clienttypes.Height{}, err
	case len(txs) > 1:
		err = fmt.Errorf("more than one transaction returned with query")
		span.SetStatus(otelcodes.Error, err.Error())
		return nil, clienttypes.Height{}, err
	}

	ack, err := core.FindPacketAcknowledgementFromEventsBySequence(txs[0].TxResult.Events, seq)
	if err != nil {
		span.SetStatus(otelcodes.Error, err.Error())
		return nil, clienttypes.Height{}, err
	}
	if ack == nil {
		err = fmt.Errorf("can't find the packet from events")
		span.SetStatus(otelcodes.Error, err.Error())
		return nil, clienttypes.Height{}, err
	}

	height := clienttypes.NewHeight(clienttypes.ParseChainID(c.ChainID()), uint64(txs[0].Height))

	return ack.Data(), height, nil
}

// QueryTxs returns an array of transactions given a tag
func (c *Chain) QueryTxs(ctx context.Context, maxHeight int64, page, limit int, events []string) ([]*ctypes.ResultTx, error) {
	if len(events) == 0 {
		return nil, errors.New("must declare at least one event to search")
	}

	events = append(events, fmt.Sprintf("tx.height<=%d", maxHeight))

	if page <= 0 {
		return nil, errors.New("page must greater than 0")
	}

	if limit <= 0 {
		return nil, errors.New("limit must greater than 0")
	}

	res, err := c.Client.TxSearch(ctx, strings.Join(events, " AND "), true, &page, &limit, "")
	if err != nil {
		return nil, err
	}
	return res.Txs, nil
}

func (c *Chain) QueryChannelUpgrade(ctx core.QueryContext) (*chantypes.QueryUpgradeResponse, error) {
	return c.queryChannelUpgrade(ctx.Context(), int64(ctx.Height().GetRevisionHeight()), false)
}

func (c *Chain) queryChannelUpgrade(ctx context.Context, height int64, prove bool) (chanRes *chantypes.QueryUpgradeResponse, err error) {
	if res, err := chanutils.QueryUpgrade(
		c.CLIContext(height).WithCmdContext(ctx),
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
	return c.queryChannelUpgradeError(ctx.Context(), int64(ctx.Height().GetRevisionHeight()), false)
}

func (c *Chain) queryChannelUpgradeError(ctx context.Context, height int64, prove bool) (chanRes *chantypes.QueryUpgradeErrorResponse, err error) {
	if res, err := chanutils.QueryUpgradeError(
		c.CLIContext(height).WithCmdContext(ctx),
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

func (c *Chain) QueryCanTransitionToFlushComplete(ctx core.QueryContext) (bool, error) {
	return c.queryCanTransitionToFlushComplete(ctx.Context(), int64(ctx.Height().GetRevisionHeight()))
}

func (c *Chain) queryCanTransitionToFlushComplete(ctx context.Context, height int64) (bool, error) {
	queryClient := chantypes.NewQueryClient(c.CLIContext(height).WithCmdContext(ctx))
	req := chantypes.QueryPacketCommitmentsRequest{
		PortId:    c.PathEnd.PortID,
		ChannelId: c.PathEnd.ChannelID,
	}
	if res, err := queryClient.PacketCommitments(ctx, &req); err != nil {
		return false, err
	} else {
		return len(res.Commitments) == 0, nil
	}
}

/////////////////////////////////////
//    STAKING -> HistoricalInfo     //
/////////////////////////////////////

// QueryHistoricalInfo returns historical header data
func (c *Chain) QueryHistoricalInfo(ctx context.Context, height clienttypes.Height) (*stakingtypes.QueryHistoricalInfoResponse, error) {
	ctx, span := tracer.Start(ctx, "Chain.QueryHistoricalInfo", core.WithChainAttributes(c.ChainID()))
	defer span.End()

	//TODO: use epoch number in query once SDK gets updated
	qc := stakingtypes.NewQueryClient(c.CLIContext(int64(height.GetRevisionHeight())).WithCmdContext(ctx))
	resp, err := qc.HistoricalInfo(ctx, &stakingtypes.QueryHistoricalInfoRequest{
		Height: int64(height.GetRevisionHeight()),
	})
	if err != nil {
		span.SetStatus(otelcodes.Error, err.Error())
		return nil, err
	}
	return resp, nil
}

// QueryValsetAtHeight returns the validator set at a given height
func (c *Chain) QueryValsetAtHeight(ctx context.Context, height clienttypes.Height) (*tmproto.ValidatorSet, error) {
	ctx, span := tracer.Start(ctx, "Chain.QueryValsetAtHeight", core.WithChainAttributes(c.ChainID()))
	defer span.End()

	res, err := c.QueryHistoricalInfo(ctx, height)
	if err != nil {
		span.SetStatus(otelcodes.Error, err.Error())
		return nil, err
	}

	// create tendermint ValidatorSet from SDK Validators
	tmVals, err := c.toTmValidators(res.Hist.Valset)
	if err != nil {
		span.SetStatus(otelcodes.Error, err.Error())
		return nil, err
	}

	sort.Sort(tmtypes.ValidatorsByVotingPower(tmVals))
	tmValSet := &tmtypes.ValidatorSet{
		Validators: tmVals,
	}
	tmValSet.GetProposer()

	protoValSet, err := tmValSet.ToProto()
	if err != nil {
		span.SetStatus(otelcodes.Error, err.Error())
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
func (c *Chain) QueryUnbondingPeriod(ctx context.Context) (time.Duration, error) {
	ctx, span := tracer.Start(ctx, "Chain.QueryUnbondingPeriod", core.WithChainAttributes(c.ChainID()))
	defer span.End()

	req := stakingtypes.QueryParamsRequest{}

	queryClient := stakingtypes.NewQueryClient(c.CLIContext(0).WithCmdContext(ctx))

	res, err := queryClient.Params(ctx, &req)
	if err != nil {
		span.SetStatus(otelcodes.Error, err.Error())
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
	return []string{fmt.Sprintf("%s.packet_dst_channel='%s'", rpTag, channelID), fmt.Sprintf("%s.packet_sequence='%d'", rpTag, seq)}
}

func writeAckQuery(channelID string, seq int) []string {
	return []string{fmt.Sprintf("%s.packet_dst_channel='%s'", waTag, channelID), fmt.Sprintf("%s.packet_sequence='%d'", waTag, seq)}
}

func channelUpgradeErrorQuery(channelID string, upgradeSequence uint64) []string {
	return []string{
		fmt.Sprintf("%s.%s='%s'",
			chantypes.EventTypeChannelUpgradeError,
			chantypes.AttributeKeyChannelID,
			channelID,
		),
		fmt.Sprintf("%s.%s='%d'",
			chantypes.EventTypeChannelUpgradeError,
			chantypes.AttributeKeyUpgradeSequence,
			upgradeSequence,
		),
	}
}
