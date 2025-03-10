package core

import (
	"testing"
	"go.uber.org/mock/gomock"

	"time"
	"context"
	"os"
	"github.com/hyperledger-labs/yui-relayer/log"
)

func TestServe(t *testing.T) {
	log.InitLoggerWithWriter("debug", "text", os.Stdout)

	ctrl := gomock.NewController(t)

	st := NewMockStrategyI(ctrl)
	sh := NewMockSyncHeaders(ctrl)
	srcChain := NewMockChain(ctrl)
	srcProver := NewMockProver(ctrl)
	dstChain := NewMockChain(ctrl)
	dstProver := NewMockProver(ctrl)

	src := NewProvableChain(srcChain, srcProver)
	dst := NewProvableChain(dstChain, dstProver)
	srv := NewRelayService(st, src, dst, sh, time.Minute, time.Minute, 3, time.Minute, 3)

	srcChain.EXPECT().ChainID().Return("srcChain").AnyTimes()
	srcChain.EXPECT().Path().Return(&PathEnd{
		ChainID: "srcChain",
		ClientID: "srcClient",
		ConnectionID: "srcConn",
		ChannelID: "srcChan",
		PortID: "srcPort",
		Order: "ORDERED",
		Version: "srcVersion",
	}).AnyTimes()

	dstChain.EXPECT().ChainID().Return("dstChain").AnyTimes()
	dstChain.EXPECT().Path().Return(&PathEnd{
		ChainID: "dstChain",
		ClientID: "dstClient",
		ConnectionID: "dstConn",
		ChannelID: "dstChan",
		PortID: "dstPort",
		Order: "ORDERED",
		Version: "dstVersion",
	}).AnyTimes()

	sh.EXPECT().Updates(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	st.EXPECT().UnrelayedPackets(gomock.Any(), gomock.Any(), gomock.Any(), false).Return(
		&RelayPackets{
			Src: []*PacketInfo{
			},
			Dst: []*PacketInfo{
			},
		}, nil,
	)

	st.EXPECT().UnrelayedAcknowledgements(gomock.Any(), gomock.Any(), gomock.Any(), false).Return(
		&RelayPackets{
			Src: []*PacketInfo{
			},
			Dst: []*PacketInfo{
			},
		}, nil,
	)

	st.EXPECT().UpdateClients(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), true).Return(
		NewRelayMsgs(), nil,
	)

	st.EXPECT().RelayPackets(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(
		NewRelayMsgs(), nil,
	)

	st.EXPECT().RelayAcknowledgements(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(
		NewRelayMsgs(), nil,
	)

	st.EXPECT().Send(gomock.Any(), gomock.Any(), gomock.Any())

	srv.Serve(context.TODO())
}
