package tendermint

import (
	"context"
	fmt "fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	querytypes "github.com/cosmos/cosmos-sdk/types/query"
	bankTypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	transfertypes "github.com/cosmos/cosmos-sdk/x/ibc/applications/transfer/types"
	clienttypes "github.com/cosmos/cosmos-sdk/x/ibc/core/02-client/types"
	conntypes "github.com/cosmos/cosmos-sdk/x/ibc/core/03-connection/types"
	chantypes "github.com/cosmos/cosmos-sdk/x/ibc/core/04-channel/types"
	ibcexported "github.com/cosmos/cosmos-sdk/x/ibc/core/exported"
	"github.com/datachainlab/relayer/core"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
)

// QueryLatestHeight queries the chain for the latest height and returns it
func (c *Chain) QueryLatestHeight() (int64, error) {
	return c.base.QueryLatestHeight()
}

// QueryValsetAtHeight returns the validator set at a given height
func (c *Chain) QueryValsetAtHeight(height clienttypes.Height) (*tmproto.ValidatorSet, error) {
	return c.base.QueryValsetAtHeight(height)
}

// QueryClientState retrevies the latest consensus state for a client in state at a given height
func (c *Chain) QueryClientState(height int64, prove bool) (*clienttypes.QueryClientStateResponse, error) {
	// TODO use arg `prove` to call the method
	return c.base.QueryClientState(height)
}

// QueryConnection returns the remote end of a given connection
func (c *Chain) QueryConnection(height int64, prove bool) (*conntypes.QueryConnectionResponse, error) {
	// TODO use arg `prove` to call the method
	return c.base.QueryConnection(height)
}

// QueryChannel returns the channel associated with a channelID
func (c *Chain) QueryChannel(height int64, prove bool) (*chantypes.QueryChannelResponse, error) {
	// TODO use arg `prove` to call the method
	return c.base.QueryChannel(height)
}

// QueryClientConsensusState retrevies the latest consensus state for a client in state at a given height
func (c *Chain) QueryClientConsensusState(height int64, dstClientConsHeight ibcexported.Height, prove bool) (*clienttypes.QueryConsensusStateResponse, error) {
	// TODO use arg `prove` to call the method
	return c.base.QueryClientConsensusState(height, dstClientConsHeight)
}

// QueryBalance returns the amount of coins in the relayer account
func (c *Chain) QueryBalance(addr sdk.AccAddress) (sdk.Coins, error) {
	params := bankTypes.NewQueryAllBalancesRequest(addr, &querytypes.PageRequest{
		Key:        []byte(""),
		Offset:     0,
		Limit:      1000,
		CountTotal: true,
	})

	queryClient := bankTypes.NewQueryClient(c.base.CLIContext(0))

	res, err := queryClient.AllBalances(context.Background(), params)
	if err != nil {
		return nil, err
	}

	return res.Balances, nil
}

// QueryDenomTraces returns all the denom traces from a given chain
func (c *Chain) QueryDenomTraces(offset, limit uint64, height int64) (*transfertypes.QueryDenomTracesResponse, error) {
	return c.base.QueryDenomTraces(offset, limit, height)
}

// QueryPacketCommitment returns the packet commitment proof at a given height
func (c *Chain) QueryPacketCommitment(height int64, seq uint64) (comRes *chantypes.QueryPacketCommitmentResponse, err error) {
	return c.base.QueryPacketCommitment(height, seq)
}

// QueryPacketAcknowledgementCommitment returns the packet ack proof at a given height
func (c *Chain) QueryPacketAcknowledgementCommitment(height int64, seq uint64) (ackRes *chantypes.QueryPacketAcknowledgementResponse, err error) {
	return c.base.QueryPacketAcknowledgement(height, seq)
}

// QueryPacketCommitments returns an array of packet commitments
func (c *Chain) QueryPacketCommitments(offset, limit, height uint64) (comRes *chantypes.QueryPacketCommitmentsResponse, err error) {
	return c.base.QueryPacketCommitments(offset, limit, height)
}

// QueryUnrecievedPackets returns a list of unrelayed packet commitments
func (c *Chain) QueryUnrecievedPackets(height uint64, seqs []uint64) ([]uint64, error) {
	return c.base.QueryUnrecievedPackets(height, seqs)
}

func (c *Chain) QueryPacketAcknowledgements(offset, limit, height uint64) (comRes *chantypes.QueryPacketAcknowledgementsResponse, err error) {
	return c.base.QueryPacketAcknowledgements(offset, limit, height)
}

func (c *Chain) QueryUnrecievedAcknowledgements(height uint64, seqs []uint64) ([]uint64, error) {
	return c.base.QueryUnrecievedAcknowledgements(height, seqs)
}

func (src *Chain) QueryPacket(height int64, seq uint64) (*chantypes.Packet, error) {
	txs, err := src.base.QueryTxs(uint64(height), 1, 1000, rcvPacketQuery(src.Path().ChannelID, int(seq)))
	switch {
	case err != nil:
		return nil, err
	case len(txs) == 0:
		return nil, fmt.Errorf("no transactions returned with query")
	case len(txs) > 1:
		return nil, fmt.Errorf("more than one transaction returned with query")
	}

	packets, err := core.GetPacketsFromEvents(txs[0].TxResult.Events)
	if err != nil {
		return nil, err
	}
	if l := len(packets); l != 1 {
		return nil, fmt.Errorf("unexpected packets length: %v", l)
	}
	return &packets[0], nil
}

func (dst *Chain) QueryPacketAcknowledgement(height int64, sequence uint64) ([]byte, error) {
	txs, err := dst.base.QueryTxs(uint64(height), 1, 1000, ackPacketQuery(dst.Path().ChannelID, int(sequence)))
	switch {
	case err != nil:
		return nil, err
	case len(txs) == 0:
		return nil, fmt.Errorf("no transactions returned with query")
	case len(txs) > 1:
		return nil, fmt.Errorf("more than one transaction returned with query")
	}

	acks, err := core.GetPacketAcknowledgementsFromEvents(txs[0].TxResult.Events)
	if err != nil {
		return nil, err
	}
	if l := len(acks); l != 1 {
		return nil, fmt.Errorf("unexpected acks length: %v", l)
	}
	return acks[0].Data(), nil
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
