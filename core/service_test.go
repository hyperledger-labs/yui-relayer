package core_test

import (
	"testing"
	"go.uber.org/mock/gomock"

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
	"github.com/hyperledger-labs/yui-relayer/metrics"
	"github.com/hyperledger-labs/yui-relayer/chains/tendermint"
)

type NaiveStrategyWrap struct {
	Inner *core.NaiveStrategy

	UnrelayedPacketsOut *core.RelayPackets
	UnrelayedAcknowledgementsOut *core.RelayPackets
	RelayPacketsOut *core.RelayMsgs
	RelayAcknowledgementsOut *core.RelayMsgs
	UpdateClientsOut *core.RelayMsgs
	SendIn *core.RelayMsgs
}
func (s *NaiveStrategyWrap) GetType() string { return s.Inner.GetType() }
func (s *NaiveStrategyWrap) SetupRelay(ctx context.Context, src, dst *core.ProvableChain) error { return s.Inner.SetupRelay(ctx, src, dst) }
func (s *NaiveStrategyWrap) UnrelayedPackets(src, dst *core.ProvableChain, sh core.SyncHeaders, includeRelayedButUnfinalized bool) (*core.RelayPackets, error) {
	ret, err := s.Inner.UnrelayedPackets(src, dst, sh, includeRelayedButUnfinalized)
	s.UnrelayedPacketsOut = ret
	return ret, err
}
func (s *NaiveStrategyWrap) RelayPackets(src, dst *core.ProvableChain, rp *core.RelayPackets, sh core.SyncHeaders, doExecuteRelaySrc, doExecuteRelayDst bool) (*core.RelayMsgs, error) {
	ret, err := s.Inner.RelayPackets(src, dst, rp, sh, doExecuteRelaySrc, doExecuteRelayDst)
	s.RelayPacketsOut = ret
	return ret, err
}
func (s *NaiveStrategyWrap) UnrelayedAcknowledgements(src, dst *core.ProvableChain, sh core.SyncHeaders, includeRelayedButUnfinalized bool) (*core.RelayPackets, error) {
	ret, err := s.Inner.UnrelayedAcknowledgements(src, dst, sh, includeRelayedButUnfinalized)
	s.UnrelayedAcknowledgementsOut = ret
	return ret, err
}
func (s *NaiveStrategyWrap) RelayAcknowledgements(src, dst *core.ProvableChain, rp *core.RelayPackets, sh core.SyncHeaders, doExecuteAckSrc, doExecuteAckDst bool) (*core.RelayMsgs, error) {
	ret, err := s.Inner.RelayAcknowledgements(src, dst, rp, sh, doExecuteAckSrc, doExecuteAckDst)
	s.RelayAcknowledgementsOut = ret
	return ret, err
}
func (s *NaiveStrategyWrap) UpdateClients(src, dst *core.ProvableChain, doExecuteRelaySrc, doExecuteRelayDst, doExecuteAckSrc, doExecuteAckDst bool, sh core.SyncHeaders, doRefresh bool) (*core.RelayMsgs, error) {
	ret, err := s.Inner.UpdateClients(src, dst, doExecuteRelaySrc, doExecuteRelayDst, doExecuteAckSrc, doExecuteAckDst, sh, doRefresh)
	fmt.Printf("UpdateClients: %v, %v, %v, %v, %v\n", doExecuteRelaySrc, doExecuteRelayDst, doExecuteAckSrc, doExecuteAckDst, doRefresh)
	s.UpdateClientsOut = ret
	return ret, err
}
func (s *NaiveStrategyWrap) Send(src, dst core.Chain, msgs *core.RelayMsgs) {
	s.SendIn = msgs
	s.Inner.Send(src, dst, msgs)
}

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
	chain.EXPECT().QueryUnfinalizedRelayPackets(gomock.Any(), gomock.Any()).Return(unfinalizedRelayPackets, nil)
	chain.EXPECT().QueryUnreceivedPackets(gomock.Any(), gomock.Any()).Return(unreceivedPackets, nil).AnyTimes()
	chain.EXPECT().QueryUnfinalizedRelayAcknowledgements(gomock.Any(), gomock.Any()).Return([]*core.PacketInfo{}, nil).AnyTimes()
	chain.EXPECT().QueryUnreceivedAcknowledgements(gomock.Any(), gomock.Any()).Return([]uint64{}, nil).AnyTimes()
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
	dstOptimizeCount uint64
	UnfinalizedRelayPackets core.PacketInfoList
	ExpectSend []string
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
			^uint64(0),
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
			[]string{
			},
		},
		"single": {
			"ORDERED",
			1,
			[]*core.PacketInfo{
				newPacketInfo(1, 99999),
			},
			[]string{
				"MsgUpdateClient(dstClient)",
				"MsgRecvPacket(1)",
			},
		},
		"queued": {
			"ORDERED",
			9,
			[]*core.PacketInfo{
				newPacketInfo(1, 99999),
			},
			[]string{
			},
		},
	}
	for n, c := range cases {
		if n[0] == '_' { continue }
		t.Run(n, func (t2 *testing.T) { testServe(t2, c) })
	}
}

func testServe(t *testing.T, tc testCase) {
	log.InitLoggerWithWriter("debug", "text", os.Stdout)
	metrics.InitializeMetrics(metrics.ExporterNull{})

	srcLatestHeader := mocktypes.Header{
		Height: clienttypes.NewHeight(10, 99),
		Timestamp: uint64(10099),
	}
	dstLatestHeader := mocktypes.Header{
		Height: clienttypes.NewHeight(110, 199),
		Timestamp: uint64(10199),
	}

	ctrl := gomock.NewController(t)

	var unreceivedPackets []uint64
	for _, p := range tc.UnfinalizedRelayPackets {
		unreceivedPackets = append(unreceivedPackets, p.Sequence)
	}
	src := NewMockProvableChain(ctrl, "src", tc.Order, srcLatestHeader, tc.UnfinalizedRelayPackets, []uint64{})
	dst := NewMockProvableChain(ctrl, "dst", tc.Order, dstLatestHeader, []*core.PacketInfo{}, unreceivedPackets)

	st := &NaiveStrategyWrap{ Inner: core.NewNaiveStrategy(false, false) }
	sh, err := core.NewSyncHeaders(src, dst)
	if err != nil {
		fmt.Printf("NewSyncHeders: %v\n", err)
	}
	var forever time.Duration = 1 << 63 - 1
	srv := core.NewRelayService(st, src, dst, sh, time.Minute, forever, 1, forever, tc.dstOptimizeCount)

	srv.Serve(context.TODO())
	fmt.Printf("UnrelayedPackets: %v\n", st.UnrelayedPacketsOut)
	fmt.Printf("UnrelayedAcknowledgementsOut: %v\n", st.UnrelayedAcknowledgementsOut)
	fmt.Printf("RelayPacketsOut: %v\n", st.RelayPacketsOut)
	fmt.Printf("RelayAcknowledgementsOut: %v\n", st.RelayAcknowledgementsOut)
	fmt.Printf("UpdateClientsOut: %v\n", st.UpdateClientsOut)
	fmt.Printf("Send.Src: %v\n", st.SendIn.Src)
	fmt.Printf("Send.Dst: %v\n", st.SendIn.Dst)
	if len(st.SendIn.Dst) != len(tc.ExpectSend) {
		t.Fatal(fmt.Sprintf("sendExpect size mismatch: %v but %v", len(tc.ExpectSend), len(st.SendIn.Dst)))
	}
	for i, msg := range st.SendIn.Dst {
		//fmt.Printf("  %v: %v\n", i, proto.MessaageName(msg))
		typeof := reflect.TypeOf(msg).Elem().Name()
		var desc string
		switch typeof {
		case "MsgUpdateClient":
			m := msg.(*clienttypes.MsgUpdateClient)
			desc = fmt.Sprintf("%s(%s)", typeof, m.ClientId)
		case"MsgRecvPacket":
			m := msg.(*chantypes.MsgRecvPacket)
			desc = fmt.Sprintf("%s(%v)", typeof, m.Packet.GetSequence())
		case"MsgTimeout":
			m := msg.(*chantypes.MsgTimeout)
			desc = fmt.Sprintf("%s(%v)", typeof, m.Packet.GetSequence())
		default:
			desc = fmt.Sprintf("%s()", typeof)
		}
		if desc != tc.ExpectSend[i] {
			t.Fatal(fmt.Sprintf("send mismatch at %v: '%s' but '%s'", i, tc.ExpectSend[i], desc))
		}
	}
}
