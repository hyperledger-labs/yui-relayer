package core_test

import (
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"testing"

	"context"
	"fmt"
	"os"
	"reflect"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	chantypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"
	mocktypes "github.com/datachainlab/ibc-mock-client/modules/light-clients/xx-mock/types"

	"github.com/hyperledger-labs/yui-relayer/chains/tendermint"
	"github.com/hyperledger-labs/yui-relayer/core"
	"github.com/hyperledger-labs/yui-relayer/internal/telemetry"
	"github.com/hyperledger-labs/yui-relayer/log"
	"github.com/hyperledger-labs/yui-relayer/provers/mock"
)

type NaiveStrategyWrap struct {
	inner *core.NaiveStrategy

	unrelayedPacketsOut          *core.RelayPackets
	processTimeoutPacketsOut     *core.RelayPackets
	unrelayedAcknowledgementsOut *core.RelayPackets
	relayPacketsOut              *core.RelayMsgs
	relayAcknowledgementsOut     *core.RelayMsgs
	updateClientsOut             *core.RelayMsgs
	sendInSrc                    []string
	sendInDst                    []string
}

func (s *NaiveStrategyWrap) GetType() string { return s.inner.GetType() }
func (s *NaiveStrategyWrap) SetupRelay(ctx context.Context, src, dst *core.ProvableChain) error {
	return s.inner.SetupRelay(ctx, src, dst)
}
func (s *NaiveStrategyWrap) UnrelayedPackets(ctx context.Context, src, dst *core.ProvableChain, sh core.SyncHeaders, includeRelayedButUnfinalized bool) (*core.RelayPackets, error) {
	ret, err := s.inner.UnrelayedPackets(ctx, src, dst, sh, includeRelayedButUnfinalized)
	s.unrelayedPacketsOut = ret
	return ret, err
}

func (s *NaiveStrategyWrap) ProcessTimeoutPackets(ctx context.Context, src, dst *core.ProvableChain, sh core.SyncHeaders, rp *core.RelayPackets) error {
	err := s.inner.ProcessTimeoutPackets(ctx, src, dst, sh, rp)
	s.processTimeoutPacketsOut = rp
	return err
}

func (s *NaiveStrategyWrap) RelayPackets(ctx context.Context, src, dst *core.ProvableChain, rp *core.RelayPackets, sh core.SyncHeaders, doExecuteRelaySrc, doExecuteRelayDst bool) (*core.RelayMsgs, error) {
	ret, err := s.inner.RelayPackets(ctx, src, dst, rp, sh, doExecuteRelaySrc, doExecuteRelayDst)
	s.relayPacketsOut = ret
	return ret, err
}
func (s *NaiveStrategyWrap) UnrelayedAcknowledgements(ctx context.Context, src, dst *core.ProvableChain, sh core.SyncHeaders, includeRelayedButUnfinalized bool) (*core.RelayPackets, error) {
	ret, err := s.inner.UnrelayedAcknowledgements(ctx, src, dst, sh, includeRelayedButUnfinalized)
	s.unrelayedAcknowledgementsOut = ret
	return ret, err
}
func (s *NaiveStrategyWrap) RelayAcknowledgements(ctx context.Context, src, dst *core.ProvableChain, rp *core.RelayPackets, sh core.SyncHeaders, doExecuteAckSrc, doExecuteAckDst bool) (*core.RelayMsgs, error) {
	ret, err := s.inner.RelayAcknowledgements(ctx, src, dst, rp, sh, doExecuteAckSrc, doExecuteAckDst)
	s.relayAcknowledgementsOut = ret
	return ret, err
}
func (s *NaiveStrategyWrap) UpdateClients(ctx context.Context, src, dst *core.ProvableChain, doExecuteRelaySrc, doExecuteRelayDst, doExecuteAckSrc, doExecuteAckDst bool, sh core.SyncHeaders, doRefresh bool) (*core.RelayMsgs, error) {
	ret, err := s.inner.UpdateClients(ctx, src, dst, doExecuteRelaySrc, doExecuteRelayDst, doExecuteAckSrc, doExecuteAckDst, sh, doRefresh)
	s.updateClientsOut = ret
	return ret, err
}
func (s *NaiveStrategyWrap) Send(ctx context.Context, src, dst core.Chain, msgs *core.RelayMsgs) {
	// format message object as string to be easily comparable
	format := func(msgs []sdk.Msg) []string {
		ret := []string{}
		for _, msg := range msgs {
			var desc string
			switch m := msg.(type) {
			case *clienttypes.MsgUpdateClient:
				desc = fmt.Sprintf("MsgUpdateClient(%s)", m.ClientId)
			case *chantypes.MsgRecvPacket:
				desc = fmt.Sprintf("MsgRecvPacket(%v)", m.Packet.GetSequence())
			case *chantypes.MsgTimeout:
				desc = fmt.Sprintf("MsgTimeout(%v)", m.Packet.GetSequence())
			default:
				desc = fmt.Sprintf("%s()", reflect.TypeOf(msg).Elem().Name())
			}
			ret = append(ret, desc)
		}
		return ret
	}
	s.sendInSrc = format(msgs.Src)
	s.sendInDst = format(msgs.Dst)
	s.inner.Send(ctx, src, dst, msgs)
}

/**
 * create mock ProvableChain with our MockProver and gomock's MockChain.
 * about height:
 *   LatestHeight: 100
 *     NextSequenceRecv: 20
 *   LatestFinalizedHeight: 90
 *     NextSequenceRecv: 10
 *   Timestamp: height + 10000
 */
var _CHAIN_STATE = struct {
	latestHeader  mocktypes.Header
	finalityDelay uint64
	sequenceRecvs map[uint64]uint64
}{
	latestHeader: mocktypes.Header{
		Height:    clienttypes.NewHeight(1, 100),
		Timestamp: uint64(10100),
	},
	finalityDelay: 10,
	sequenceRecvs: map[uint64]uint64{ //note that nextSequenceRecv is +1
		100: 20,
		90:  10,
	},
}

func NewMockProvableChain(
	ctrl *gomock.Controller,
	name, order string,
	unfinalizedRelayPackets core.PacketInfoList,
	unreceivedPackets []uint64,
) *core.ProvableChain {
	chain := core.NewMockChain(ctrl)
	prover := mock.NewProver(chain, mock.ProverConfig{FinalityDelay: _CHAIN_STATE.finalityDelay})

	chain.EXPECT().ChainID().Return(name + "Chain").AnyTimes()
	chain.EXPECT().Codec().Return(nil).AnyTimes()
	chain.EXPECT().GetAddress().Return(sdk.AccAddress{}, nil).AnyTimes()
	chain.EXPECT().Path().Return(&core.PathEnd{
		ChainID:      name + "Chain",
		ClientID:     name + "Client",
		ConnectionID: name + "Conn",
		ChannelID:    name + "Chan",
		PortID:       name + "Port",
		Order:        order,
		Version:      name + "Version",
	}).AnyTimes()
	chain.EXPECT().LatestHeight(gomock.Any()).Return(_CHAIN_STATE.latestHeader.Height, nil).AnyTimes()
	chain.EXPECT().Timestamp(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, h ibcexported.Height) (time.Time, error) {
			return time.Unix(0, int64(10000+h.GetRevisionHeight())), nil
		}).AnyTimes()
	chain.EXPECT().QueryNextSequenceReceive(gomock.Any()).DoAndReturn(
		func(ctx core.QueryContext) (*chantypes.QueryNextSequenceReceiveResponse, error) {
			// get most recent sequence earlier than targetHeight
			var lastHeight uint64 = 0
			var lastSequence uint64 = 0
			for h, s := range _CHAIN_STATE.sequenceRecvs {
				if h <= ctx.Height().GetRevisionHeight() && lastHeight < h {
					lastHeight = h
					lastSequence = s
				}
			}
			return &chantypes.QueryNextSequenceReceiveResponse{
				NextSequenceReceive: lastSequence + 1,
				Proof:               []byte{},
				ProofHeight:         ctx.Height().(clienttypes.Height),
			}, nil
		}).AnyTimes()
	chain.EXPECT().QueryUnfinalizedRelayPackets(gomock.Any(), gomock.Any()).Return(unfinalizedRelayPackets, nil).AnyTimes()
	chain.EXPECT().QueryUnreceivedPackets(gomock.Any(), gomock.Any()).Return(unreceivedPackets, nil).AnyTimes()
	chain.EXPECT().QueryUnreceivedAcknowledgements(gomock.Any(), gomock.Any()).Return([]uint64{}, nil).AnyTimes()
	chain.EXPECT().QueryUnfinalizedRelayAcknowledgements(gomock.Any(), gomock.Any()).Return([]*core.PacketInfo{}, nil).AnyTimes()
	chain.EXPECT().SendMsgs(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, msgs []sdk.Msg) ([]core.MsgID, error) {
		var msgIDs []core.MsgID
		for _, _ = range msgs {
			msgIDs = append(msgIDs, &tendermint.MsgID{TxHash: "", MsgIndex: 0})
		}
		return msgIDs, nil
	}).AnyTimes()
	return core.NewProvableChain(chain, prover)
}

type testCase struct {
	order                      string
	optimizeCount              uint64
	unfinalizedRelayPacketsSrc core.PacketInfoList
	unfinalizedRelayPacketsDst core.PacketInfoList
	expectSendSrc              []string
	expectSendDst              []string
}

func newPacketInfo(seq uint64, timeoutHeight uint64) *core.PacketInfo {
	return &core.PacketInfo{
		Packet: chantypes.NewPacket(
			[]byte{},
			seq,
			"srcPort",
			"srcChannel",
			"dstPort",
			"dstChannel",
			clienttypes.NewHeight(1, timeoutHeight),
			0, // timeoutTimestamp
		),
		EventHeight: clienttypes.NewHeight(1, 1),
	}
}

func TestServe(t *testing.T) {
	cases := map[string]testCase{
		"empty": {
			"ORDERED",
			1,
			[]*core.PacketInfo{},
			[]*core.PacketInfo{},
			[]string{},
			[]string{},
		},
		"single": { // all src packets are relayed to dst with leading UpdateClient message
			"ORDERED",
			1,
			[]*core.PacketInfo{
				newPacketInfo(1, 9999), // note that nextSequenceRecv is not checked in relaying normal packets
			},
			[]*core.PacketInfo{},
			[]string{},
			[]string{
				"MsgUpdateClient(dstClient)",
				"MsgRecvPacket(1)",
			},
		},
		"multi": { // multiple packets. The rest is the same as the "single" case.
			"ORDERED",
			1,
			[]*core.PacketInfo{
				newPacketInfo(1, 9999),
				newPacketInfo(2, 9999),
				newPacketInfo(3, 9999),
			},
			[]*core.PacketInfo{},
			[]string{},
			[]string{
				"MsgUpdateClient(dstClient)",
				"MsgRecvPacket(1)",
				"MsgRecvPacket(2)",
				"MsgRecvPacket(3)",
			},
		},
		"queued": { // packets less than optimizeCount are queed and not relayed
			"ORDERED",
			9,
			[]*core.PacketInfo{
				newPacketInfo(1, 9999),
			},
			[]*core.PacketInfo{},
			[]string{},
			[]string{},
		},
		"not timeout(at border height)": { // An packet which timeouted at 101 are normally relayed at 100th block.
			"ORDERED",
			1,
			[]*core.PacketInfo{
				newPacketInfo(1, 101),
			},
			[]*core.PacketInfo{},
			[]string{},
			[]string{
				"MsgUpdateClient(dstClient)",
				"MsgRecvPacket(1)",
			},
		},
		"timeout and previous packet is finalized": { // timeout. Relay back to src channel as MsgTimeout with UpdateClient.
			"ORDERED",
			1,
			[]*core.PacketInfo{
				// timeout height of the packet is 90 and latest finalized height is 90. So it is timed out.
				// nextSequenceRecv at finalized height(=90) is 11 in _STATE_CHAIN config.
				newPacketInfo(11, 90),
			},
			[]*core.PacketInfo{},
			[]string{"MsgUpdateClient(srcClient)", "MsgTimeout(11)"},
			[]string{},
		},
		"timeout but previous packet is not finalized": {
			"ORDERED",
			1,
			[]*core.PacketInfo{
				newPacketInfo(12, 90),
			},
			[]*core.PacketInfo{},
			[]string{},
			[]string{},
		},
		"timeout at latest block but not at finalized block(at lower border)": { // waiting relay in finalized block
			"ORDERED",
			1,
			[]*core.PacketInfo{
				newPacketInfo(11, 91),
			},
			[]*core.PacketInfo{},
			[]string{},
			[]string{},
		},
		"timeout at latest block but not at finalized block(at heigher border)": { // waiting relay in finalized block
			"ORDERED",
			1,
			[]*core.PacketInfo{
				newPacketInfo(11, 100),
			},
			[]*core.PacketInfo{},
			[]string{},
			[]string{},
		},
		"multiple timeouts packets in ordered channel": { // In ordered channel, later packets from timeouted packets are not relayerd
			"ORDERED",
			1,
			[]*core.PacketInfo{
				newPacketInfo(11, 9),
				newPacketInfo(12, 9999),
				newPacketInfo(13, 9),
			},
			[]*core.PacketInfo{},
			[]string{
				"MsgUpdateClient(srcClient)",
				"MsgTimeout(11)",
			},
			[]string{},
		},
		"relay preceding packets before timeouted one": { // In ordered channel, only preceding packets before timeout packets are relayed.
			"ORDERED",
			1,
			[]*core.PacketInfo{
				newPacketInfo(11, 9999),
				newPacketInfo(12, 9999),
				newPacketInfo(13, 9),
			},
			[]*core.PacketInfo{},
			[]string{},
			[]string{
				"MsgUpdateClient(dstClient)",
				"MsgRecvPacket(11)",
				"MsgRecvPacket(12)",
			},
		},
		"multiple timeouts packets in ordered channel(both side)": {
			"ORDERED",
			1,
			[]*core.PacketInfo{
				newPacketInfo(11, 9),
				newPacketInfo(12, 9999),
				newPacketInfo(13, 9),
			},
			[]*core.PacketInfo{
				newPacketInfo(11, 9999),
				newPacketInfo(12, 9999),
				newPacketInfo(13, 9),
			},
			[]string{
				"MsgUpdateClient(srcClient)",
				"MsgRecvPacket(11)",
				"MsgRecvPacket(12)",
				"MsgTimeout(11)",
			},
			[]string{},
		},
		"multiple timeout packets in unordered channel": { // In unordered channel, all timeout packets are backed and others are relayed.
			"UNORDERED",
			1,
			[]*core.PacketInfo{
				newPacketInfo(11, 9999),
				newPacketInfo(12, 9),
				newPacketInfo(13, 9999),
				newPacketInfo(14, 9),
			},
			[]*core.PacketInfo{},
			[]string{
				"MsgUpdateClient(srcClient)",
				"MsgTimeout(12)",
				"MsgTimeout(14)",
			},
			[]string{
				"MsgUpdateClient(dstClient)",
				"MsgRecvPacket(11)",
				"MsgRecvPacket(13)",
			},
		},
		"multiple timeout packets in nordered channel(both side)": { // In unordered channel, all timeout packets are backed and others are relayed.
			"UNORDERED",
			1,
			[]*core.PacketInfo{
				newPacketInfo(11, 9999),
				newPacketInfo(12, 9),
				newPacketInfo(13, 9999),
				newPacketInfo(14, 9),
			},
			[]*core.PacketInfo{
				newPacketInfo(11, 9999),
				newPacketInfo(12, 9),
				newPacketInfo(13, 9999),
				newPacketInfo(14, 9),
			},
			[]string{
				"MsgUpdateClient(srcClient)",
				"MsgRecvPacket(11)",
				"MsgRecvPacket(13)",
				"MsgTimeout(12)",
				"MsgTimeout(14)",
			},
			[]string{
				"MsgUpdateClient(dstClient)",
				"MsgRecvPacket(11)",
				"MsgRecvPacket(13)",
				"MsgTimeout(12)",
				"MsgTimeout(14)",
			},
		},
	}
	for n, c := range cases {
		if n[0] == '_' {
			continue
		}
		t.Run(n, func(t2 *testing.T) { testServe(t2, c) })
	}
}

func testServe(t *testing.T, tc testCase) {
	log.InitLoggerWithWriter("debug", "text", os.Stdout, false)
	telemetry.InitializeMetrics()

	ctrl := gomock.NewController(t)

	var unreceivedPacketsSrc, unreceivedPacketsDst []uint64
	for _, p := range tc.unfinalizedRelayPacketsSrc {
		unreceivedPacketsSrc = append(unreceivedPacketsSrc, p.Sequence)
	}
	for _, p := range tc.unfinalizedRelayPacketsDst {
		unreceivedPacketsDst = append(unreceivedPacketsDst, p.Sequence)
	}
	src := NewMockProvableChain(ctrl, "src", tc.order, tc.unfinalizedRelayPacketsSrc, unreceivedPacketsDst)
	dst := NewMockProvableChain(ctrl, "dst", tc.order, tc.unfinalizedRelayPacketsDst, unreceivedPacketsSrc)

	st := &NaiveStrategyWrap{inner: core.NewNaiveStrategy(false, false)}
	sh, err := core.NewSyncHeaders(context.TODO(), src, dst)
	if err != nil {
		fmt.Printf("NewSyncHeders: %v\n", err)
	}
	var forever time.Duration = 1<<63 - 1
	srv := core.NewRelayService(st, src, dst, sh, time.Minute, forever, tc.optimizeCount, forever, tc.optimizeCount)

	srv.Serve(context.TODO())

	t.Logf("UnrelayedPackets: %v\n", st.unrelayedPacketsOut)
	t.Logf("UnrelayedAcknowledgementsOut: %v\n", st.unrelayedAcknowledgementsOut)
	t.Logf("RelayPacketsOut: %v\n", st.relayPacketsOut)
	t.Logf("RelayAcknowledgementsOut: %v\n", st.relayAcknowledgementsOut)
	t.Logf("UpdateClientsOut: %v\n", st.updateClientsOut)
	t.Logf("Send.Src: %v\n", st.sendInSrc)
	t.Logf("Send.Dst: %v\n", st.sendInDst)

	assert.Equal(t, tc.expectSendSrc, st.sendInSrc, "Send.Src")
	assert.Equal(t, tc.expectSendDst, st.sendInDst, "Send.Dst")
}
