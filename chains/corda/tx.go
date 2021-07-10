package corda

import (
	"context"
	"log"

	sdk "github.com/cosmos/cosmos-sdk/types"
	transfertypes "github.com/cosmos/ibc-go/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	conntypes "github.com/cosmos/ibc-go/modules/core/03-connection/types"
	chantypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
)

func (c *Chain) SendMsgs(ctx context.Context, msgs []sdk.Msg) ([]byte, error) {
	for _, msg := range msgs {
		var err error
		switch msg := msg.(type) {
		case *clienttypes.MsgCreateClient:
			err = c.client.createClient(ctx, msg)
		case *conntypes.MsgConnectionOpenInit:
			err = c.client.connOpenInit(ctx, msg)
		case *conntypes.MsgConnectionOpenTry:
			err = c.client.connOpenTry(ctx, msg)
		case *conntypes.MsgConnectionOpenAck:
			err = c.client.connOpenAck(ctx, msg)
		case *conntypes.MsgConnectionOpenConfirm:
			err = c.client.connOpenConfirm(ctx, msg)
		case *chantypes.MsgChannelOpenInit:
			err = c.client.chanOpenInit(ctx, msg)
		case *chantypes.MsgChannelOpenTry:
			err = c.client.chanOpenTry(ctx, msg)
		case *chantypes.MsgChannelOpenAck:
			err = c.client.chanOpenAck(ctx, msg)
		case *chantypes.MsgChannelOpenConfirm:
			err = c.client.chanOpenConfirm(ctx, msg)
		case *chantypes.MsgRecvPacket:
			err = c.client.recvPacket(ctx, msg)
		case *chantypes.MsgAcknowledgement:
			err = c.client.acknowledgement(ctx, msg)
		case *transfertypes.MsgTransfer:
			err = c.client.transfer(ctx, msg)
		default:
			panic("illegal msg type")
		}
		if err != nil {
			return nil, err
		}
	}
	return nil, nil
}

// Send sends msgs to the chain and logging a result of it
// It returns a boolean value whether the result is success
func (c *Chain) Send(ctx context.Context, msgs []sdk.Msg) bool {
	log.Printf("corda.SendMsgs: %v", msgs)
	_, err := c.SendMsgs(ctx, msgs)
	log.Printf("corda.SendMsgs.result: err=%v", err)
	return err == nil
}

func (cic *cordaIbcClient) createClient(ctx context.Context, msg *clienttypes.MsgCreateClient) error {
	_, err := cic.clientTx.CreateClient(
		ctx,
		msg,
	)
	return err
}

func (cic *cordaIbcClient) connOpenInit(ctx context.Context, msg *conntypes.MsgConnectionOpenInit) error {
	_, err := cic.connTx.ConnectionOpenInit(
		ctx,
		msg,
	)
	return err
}

func (cic *cordaIbcClient) connOpenTry(ctx context.Context, msg *conntypes.MsgConnectionOpenTry) error {
	_, err := cic.connTx.ConnectionOpenTry(
		ctx,
		msg,
	)
	return err
}

func (cic *cordaIbcClient) connOpenAck(ctx context.Context, msg *conntypes.MsgConnectionOpenAck) error {
	_, err := cic.connTx.ConnectionOpenAck(
		ctx,
		msg,
	)
	return err
}

func (cic *cordaIbcClient) connOpenConfirm(ctx context.Context, msg *conntypes.MsgConnectionOpenConfirm) error {
	_, err := cic.connTx.ConnectionOpenConfirm(
		ctx,
		msg,
	)
	return err
}

func (cic *cordaIbcClient) chanOpenInit(ctx context.Context, msg *chantypes.MsgChannelOpenInit) error {
	_, err := cic.chanTx.ChannelOpenInit(
		ctx,
		msg,
	)
	return err
}

func (cic *cordaIbcClient) chanOpenTry(ctx context.Context, msg *chantypes.MsgChannelOpenTry) error {
	_, err := cic.chanTx.ChannelOpenTry(
		ctx,
		msg,
	)
	return err
}

func (cic *cordaIbcClient) chanOpenAck(ctx context.Context, msg *chantypes.MsgChannelOpenAck) error {
	_, err := cic.chanTx.ChannelOpenAck(
		ctx,
		msg,
	)
	return err
}

func (cic *cordaIbcClient) chanOpenConfirm(ctx context.Context, msg *chantypes.MsgChannelOpenConfirm) error {
	_, err := cic.chanTx.ChannelOpenConfirm(
		ctx,
		msg,
	)
	return err
}

func (cic *cordaIbcClient) recvPacket(ctx context.Context, msg *chantypes.MsgRecvPacket) error {
	_, err := cic.chanTx.RecvPacket(
		ctx,
		msg,
	)
	return err
}

func (cic *cordaIbcClient) acknowledgement(ctx context.Context, msg *chantypes.MsgAcknowledgement) error {
	_, err := cic.chanTx.Acknowledgement(
		ctx,
		msg,
	)
	return err
}

func (cic *cordaIbcClient) transfer(ctx context.Context, msg *transfertypes.MsgTransfer) error {
	_, err := cic.transferTx.Transfer(
		ctx,
		msg,
	)
	return err
}
