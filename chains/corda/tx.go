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

func (c *Chain) SendMsgs(msgs []sdk.Msg) ([]byte, error) {
	for _, msg := range msgs {
		var err error
		switch msg := msg.(type) {
		case *clienttypes.MsgCreateClient:
			err = c.client.createClient(msg)
		case *conntypes.MsgConnectionOpenInit:
			err = c.client.connOpenInit(msg)
		case *conntypes.MsgConnectionOpenTry:
			err = c.client.connOpenTry(msg)
		case *conntypes.MsgConnectionOpenAck:
			err = c.client.connOpenAck(msg)
		case *conntypes.MsgConnectionOpenConfirm:
			err = c.client.connOpenConfirm(msg)
		case *chantypes.MsgChannelOpenInit:
			err = c.client.chanOpenInit(msg)
		case *chantypes.MsgChannelOpenTry:
			err = c.client.chanOpenTry(msg)
		case *chantypes.MsgChannelOpenAck:
			err = c.client.chanOpenAck(msg)
		case *chantypes.MsgChannelOpenConfirm:
			err = c.client.chanOpenConfirm(msg)
		case *chantypes.MsgRecvPacket:
			err = c.bankNodeClient.recvPacket(msg)
		case *chantypes.MsgAcknowledgement:
			err = c.client.acknowledgement(msg)
		case *transfertypes.MsgTransfer:
			err = c.client.transfer(msg)
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
func (c *Chain) Send(msgs []sdk.Msg) bool {
	log.Printf("corda.SendMsgs: %v", msgs)
	_, err := c.SendMsgs(msgs)
	log.Printf("corda.SendMsgs.result: err=%v", err)
	return err == nil
}

func (cic *cordaIbcClient) createClient(msg *clienttypes.MsgCreateClient) error {
	_, err := cic.clientTx.CreateClient(
		context.TODO(),
		msg,
	)
	return err
}

func (cic *cordaIbcClient) connOpenInit(msg *conntypes.MsgConnectionOpenInit) error {
	_, err := cic.connTx.ConnectionOpenInit(
		context.TODO(),
		msg,
	)
	return err
}

func (cic *cordaIbcClient) connOpenTry(msg *conntypes.MsgConnectionOpenTry) error {
	_, err := cic.connTx.ConnectionOpenTry(
		context.TODO(),
		msg,
	)
	return err
}

func (cic *cordaIbcClient) connOpenAck(msg *conntypes.MsgConnectionOpenAck) error {
	_, err := cic.connTx.ConnectionOpenAck(
		context.TODO(),
		msg,
	)
	return err
}

func (cic *cordaIbcClient) connOpenConfirm(msg *conntypes.MsgConnectionOpenConfirm) error {
	_, err := cic.connTx.ConnectionOpenConfirm(
		context.TODO(),
		msg,
	)
	return err
}

func (cic *cordaIbcClient) chanOpenInit(msg *chantypes.MsgChannelOpenInit) error {
	_, err := cic.chanTx.ChannelOpenInit(
		context.TODO(),
		msg,
	)
	return err
}

func (cic *cordaIbcClient) chanOpenTry(msg *chantypes.MsgChannelOpenTry) error {
	_, err := cic.chanTx.ChannelOpenTry(
		context.TODO(),
		msg,
	)
	return err
}

func (cic *cordaIbcClient) chanOpenAck(msg *chantypes.MsgChannelOpenAck) error {
	_, err := cic.chanTx.ChannelOpenAck(
		context.TODO(),
		msg,
	)
	return err
}

func (cic *cordaIbcClient) chanOpenConfirm(msg *chantypes.MsgChannelOpenConfirm) error {
	_, err := cic.chanTx.ChannelOpenConfirm(
		context.TODO(),
		msg,
	)
	return err
}

func (cic *cordaIbcClient) recvPacket(msg *chantypes.MsgRecvPacket) error {
	_, err := cic.chanTx.RecvPacket(
		context.TODO(),
		msg,
	)
	return err
}

func (cic *cordaIbcClient) acknowledgement(msg *chantypes.MsgAcknowledgement) error {
	_, err := cic.chanTx.Acknowledgement(
		context.TODO(),
		msg,
	)
	return err
}

func (cic *cordaIbcClient) transfer(msg *transfertypes.MsgTransfer) error {
	_, err := cic.transferTx.Transfer(
		context.TODO(),
		msg,
	)
	return err
}
