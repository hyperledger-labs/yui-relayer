package core_test

import (
	"testing"
	"go.uber.org/mock/gomock"
	"github.com/stretchr/testify/assert"

	"time"
	"context"
	"os"
	"fmt"
	"reflect"

	sdk "github.com/cosmos/cosmos-sdk/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	chantypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	mocktypes "github.com/datachainlab/ibc-mock-client/modules/light-clients/xx-mock/types"
	ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"

	"github.com/hyperledger-labs/yui-relayer/core"
	"github.com/hyperledger-labs/yui-relayer/log"
	"github.com/hyperledger-labs/yui-relayer/provers/mock"
	"github.com/hyperledger-labs/yui-relayer/chains/tendermint"
	"github.com/hyperledger-labs/yui-relayer/internal/telemetry"
)

type NaiveStrategyWrap struct {
	Inner *core.NaiveStrategy

	UnrelayedPacketsOut *core.RelayPackets
	ProcessTimeoutPacketsOut *core.RelayPackets
	UnrelayedAcknowledgementsOut *core.RelayPackets
	RelayPacketsOut *core.RelayMsgs
	RelayAcknowledgementsOut *core.RelayMsgs
	UpdateClientsOut *core.RelayMsgs
	SendInSrc []string
	SendInDst []string
}
func (s *NaiveStrategyWrap) GetType() string { return s.Inner.GetType() }
func (s *NaiveStrategyWrap) SetupRelay(ctx context.Context, src, dst *core.ProvableChain) error { return s.Inner.SetupRelay(ctx, src, dst) }
func (s *NaiveStrategyWrap) UnrelayedPackets(ctx context.Context, src, dst *core.ProvableChain, sh core.SyncHeaders, includeRelayedButUnfinalized bool) (*core.RelayPackets, error) {
	ret, err := s.Inner.UnrelayedPackets(ctx, src, dst, sh, includeRelayedButUnfinalized)
	s.UnrelayedPacketsOut = ret
	return ret, err
}

func (s *NaiveStrategyWrap) ProcessTimeoutPackets(ctx context.Context, src, dst *core.ProvableChain, sh core.SyncHeaders, rp *core.RelayPackets) error {
	err := s.Inner.ProcessTimeoutPackets(ctx, src, dst, sh, rp)
	s.ProcessTimeoutPacketsOut = rp
	return err
}

func (s *NaiveStrategyWrap) RelayPackets(ctx context.Context, src, dst *core.ProvableChain, rp *core.RelayPackets, sh core.SyncHeaders, doExecuteRelaySrc, doExecuteRelayDst bool) (*core.RelayMsgs, error) {
	ret, err := s.Inner.RelayPackets(ctx, src, dst, rp, sh, doExecuteRelaySrc, doExecuteRelayDst)
	s.RelayPacketsOut = ret
	return ret, err
}
func (s *NaiveStrategyWrap) UnrelayedAcknowledgements(ctx context.Context, src, dst *core.ProvableChain, sh core.SyncHeaders, includeRelayedButUnfinalized bool) (*core.RelayPackets, error) {
	ret, err := s.Inner.UnrelayedAcknowledgements(ctx, src, dst, sh, includeRelayedButUnfinalized)
	s.UnrelayedAcknowledgementsOut = ret
	return ret, err
}
func (s *NaiveStrategyWrap) RelayAcknowledgements(ctx context.Context, src, dst *core.ProvableChain, rp *core.RelayPackets, sh core.SyncHeaders, doExecuteAckSrc, doExecuteAckDst bool) (*core.RelayMsgs, error) {
	ret, err := s.Inner.RelayAcknowledgements(ctx, src, dst, rp, sh, doExecuteAckSrc, doExecuteAckDst)
	s.RelayAcknowledgementsOut = ret
	return ret, err
}
func (s *NaiveStrategyWrap) UpdateClients(ctx context.Context, src, dst *core.ProvableChain, doExecuteRelaySrc, doExecuteRelayDst, doExecuteAckSrc, doExecuteAckDst bool, sh core.SyncHeaders, doRefresh bool) (*core.RelayMsgs, error) {
	ret, err := s.Inner.UpdateClients(ctx, src, dst, doExecuteRelaySrc, doExecuteRelayDst, doExecuteAckSrc, doExecuteAckDst, sh, doRefresh)
	s.UpdateClientsOut = ret
	return ret, err
}
func (s *NaiveStrategyWrap) Send(ctx context.Context, src, dst core.Chain, msgs *core.RelayMsgs) {
	// format message object as string to be easily comparable
	format := func(msgs []sdk.Msg) ([]string) {
		ret := []string{}
		for _, msg := range msgs {
			typeof := reflect.TypeOf(msg).Elem().Name()
			var desc string
			switch typeof {
			case "MsgUpdateClient":
				m := msg.(*clienttypes.MsgUpdateClient)
				desc = fmt.Sprintf("%s(%s)", typeof, m.ClientId)
			case "MsgRecvPacket":
				m := msg.(*chantypes.MsgRecvPacket)
				desc = fmt.Sprintf("%s(%v)", typeof, m.Packet.GetSequence())
			case "MsgTimeout":
				m := msg.(*chantypes.MsgTimeout)
				desc = fmt.Sprintf("%s(%v)", typeof, m.Packet.GetSequence())
			default:
				desc = fmt.Sprintf("%s()", typeof)
			}
			ret = append(ret, desc)
		}
		return ret
	}
	s.SendInSrc = format(msgs.Src)
	s.SendInDst = format(msgs.Dst)
	s.Inner.Send(ctx, src, dst, msgs)
}

/**
  * create mock ProvableChain with our MockProver and gomock's MockChain.
  * about height:
  *   LatestHeight: returns header that NewMockProvableChain is given
  *   LatestFinalizedHeight: LatestHeight - 10
  *   Timestamp: height + 10000
  */
func NewMockProvableChain(
	ctrl *gomock.Controller,
	name, order string,
	header mocktypes.Header,
	unfinalizedRelayPackets core.PacketInfoList,
	unreceivedPackets []uint64,
) *core.ProvableChain {
	chain := core.NewMockChain(ctrl)
	prover := mock.NewProver(chain, mock.ProverConfig{ FinalityDelay: 10 })

	chain.EXPECT().ChainID().Return(name + "Chain").AnyTimes()
	chain.EXPECT().Codec().Return(nil).AnyTimes()
	chain.EXPECT().GetAddress().Return(sdk.AccAddress{}, nil).AnyTimes()
	chain.EXPECT().Path().Return(&core.PathEnd{
		ChainID: name + "Chain",
		ClientID: name + "Client",
		ConnectionID: name + "Conn",
		ChannelID: name + "Chan",
		PortID: name + "Port",
		Order: order,
		Version: name + "Version",
	}).AnyTimes()
	chain.EXPECT().LatestHeight(gomock.Any()).Return(header.Height, nil).AnyTimes()
	chain.EXPECT().Timestamp(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, h ibcexported.Height) (time.Time, error) {
			return time.Unix(0, int64(10000 + h.GetRevisionHeight())), nil
		}).AnyTimes()
	chain.EXPECT().QueryNextSequenceReceive(gomock.Any()).DoAndReturn(
		func(ctx core.QueryContext) (*chantypes.QueryNextSequenceReceiveResponse, error) {
			height := ctx.Height().(clienttypes.Height)
			return &chantypes.QueryNextSequenceReceiveResponse{ 1, []byte{}, height }, nil
		}).AnyTimes()
	chain.EXPECT().QueryUnfinalizedRelayPackets(gomock.Any(), gomock.Any()).Return(unfinalizedRelayPackets, nil).AnyTimes()
	chain.EXPECT().QueryUnreceivedPackets(gomock.Any(), gomock.Any()).Return(unreceivedPackets, nil).AnyTimes()
	chain.EXPECT().QueryUnreceivedAcknowledgements(gomock.Any(), gomock.Any()).Return([]uint64{}, nil).AnyTimes()
	chain.EXPECT().QueryUnfinalizedRelayAcknowledgements(gomock.Any(), gomock.Any()).Return([]*core.PacketInfo{}, nil).AnyTimes()
	chain.EXPECT().SendMsgs(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, msgs []sdk.Msg) ([]core.MsgID, error) {
		var msgIDs []core.MsgID
		for _, _ = range msgs {
			msgIDs = append(msgIDs, &tendermint.MsgID{TxHash:"", MsgIndex:0})
		}
		return msgIDs, nil
	}).AnyTimes()
	return core.NewProvableChain(chain, prover)
}

type testCase struct {
	Order string
	optimizeCount uint64
	UnfinalizedRelayPackets core.PacketInfoList
	ExpectSendSrc []string
	ExpectSendDst []string
}

func newPacketInfo(seq uint64, timeoutHeight uint64) (*core.PacketInfo) {
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
			[]string{ },
			[]string{ },
		},
		"single": {
			"ORDERED",
			1,
			[]*core.PacketInfo{
				newPacketInfo(1, 9999),
			},
			[]string{ },
			[]string{
				"MsgUpdateClient(dstClient)",
				"MsgRecvPacket(1)",
			},
		},
		"multi": {
			"ORDERED",
			1,
			[]*core.PacketInfo{
				newPacketInfo(1, 9999),
				newPacketInfo(2, 9999),
				newPacketInfo(3, 9999),
			},
			[]string{ },
			[]string{
				"MsgUpdateClient(dstClient)",
				"MsgRecvPacket(1)",
				"MsgRecvPacket(2)",
				"MsgRecvPacket(3)",
			},
		},
		"queued": {
			"ORDERED",
			9,
			[]*core.PacketInfo{
				newPacketInfo(1, 9999),
			},
			[]string{ },
			[]string{ },
		},
		"@not timeout(at border height)": {
			"ORDERED",
			1,
			[]*core.PacketInfo{
				newPacketInfo(1, 101),
			},
			[]string{ },
			[]string{
				"MsgUpdateClient(dstClient)",
				"MsgRecvPacket(1)",
			},
		},
		"timeout": {
			"ORDERED",
			1,
			[]*core.PacketInfo{
				newPacketInfo(1, 90),
			},
			[]string{ "MsgUpdateClient(srcClient)", "MsgTimeout(1)" },
			[]string{ },
		},
		"timeout at latest block but not at finalized block(at lower border)": {
			"ORDERED",
			1,
			[]*core.PacketInfo{
				newPacketInfo(1, 91),
			},
			[]string{ },
			[]string{ },
		},
		"timeout at latest block but not at finalized block(at heigher border)": {
			"ORDERED",
			1,
			[]*core.PacketInfo{
				newPacketInfo(1, 100),
			},
			[]string{ },
			[]string{ },
		},
		"only packets precede timeout packet": {
			"ORDERED",
			1,
			[]*core.PacketInfo{
				newPacketInfo(1, 9999),
				newPacketInfo(2, 9999),
				newPacketInfo(3, 9),
			},
			[]string{ },
			[]string{
				"MsgUpdateClient(dstClient)",
				"MsgRecvPacket(1)",
				"MsgRecvPacket(2)",
			},
		},
	}
	for n, c := range cases {
		if n[0] == '_' { continue }
		t.Run(n, func (t2 *testing.T) { testServe(t2, c) })
	}
}

func testServe(t *testing.T, tc testCase) {
	log.InitLoggerWithWriter("debug", "text", os.Stdout, false)
	telemetry.InitializeMetrics()

	srcLatestHeader := mocktypes.Header{
		Height: clienttypes.NewHeight(1, 100),
		Timestamp: uint64(10100),
	}
	dstLatestHeader := mocktypes.Header{
		Height: clienttypes.NewHeight(1, 100),
		Timestamp: uint64(10100),
	}

	ctrl := gomock.NewController(t)

	var unreceivedPackets []uint64
	for _, p := range tc.UnfinalizedRelayPackets {
		unreceivedPackets = append(unreceivedPackets, p.Sequence)
	}
	src := NewMockProvableChain(ctrl, "src", tc.Order, srcLatestHeader, tc.UnfinalizedRelayPackets, []uint64{})
	dst := NewMockProvableChain(ctrl, "dst", tc.Order, dstLatestHeader, []*core.PacketInfo{}, unreceivedPackets)

	st := &NaiveStrategyWrap{ Inner: core.NewNaiveStrategy(false, false) }
	sh, err := core.NewSyncHeaders(context.TODO(), src, dst)
	if err != nil {
		fmt.Printf("NewSyncHeders: %v\n", err)
	}
	var forever time.Duration = 1 << 63 - 1
	srv := core.NewRelayService(st, src, dst, sh, time.Minute, forever, tc.optimizeCount, forever, tc.optimizeCount)

	srv.Serve(context.TODO())

	t.Logf("UnrelayedPackets: %v\n", st.UnrelayedPacketsOut)
	t.Logf("UnrelayedAcknowledgementsOut: %v\n", st.UnrelayedAcknowledgementsOut)
	t.Logf("RelayPacketsOut: %v\n", st.RelayPacketsOut)
	t.Logf("RelayAcknowledgementsOut: %v\n", st.RelayAcknowledgementsOut)
	t.Logf("UpdateClientsOut: %v\n", st.UpdateClientsOut)
	t.Logf("Send.Src: %v\n", st.SendInSrc)
	t.Logf("Send.Dst: %v\n", st.SendInDst)

	assert.Equal(t, tc.ExpectSendSrc, st.SendInSrc, "Send.Src")
	assert.Equal(t, tc.ExpectSendDst, st.SendInDst, "Send.Dst")
}
