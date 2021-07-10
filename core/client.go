package core

import (
	"context"
	"fmt"
	"log"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func CreateClients(ctx context.Context, src, dst ChainI) error {
	var (
		clients = &RelayMsgs{Src: []sdk.Msg{}, Dst: []sdk.Msg{}}
	)

	srcH, dstH, err := UpdatesWithHeaders(ctx, src, dst)
	if err != nil {
		return err
	}

	srcAddr, err := src.GetAddress()
	if err != nil {
		return err
	}
	dstAddr, err := dst.GetAddress()
	if err != nil {
		return err
	}

	{
		msg, err := dst.MakeMsgCreateClient(src.ClientID(), dstH, srcAddr)
		if err != nil {
			return err
		}
		clients.Src = append(clients.Src, msg)
	}

	{
		msg, err := src.MakeMsgCreateClient(dst.ClientID(), srcH, dstAddr)
		if err != nil {
			return err
		}
		clients.Dst = append(clients.Dst, msg)
	}

	// Send msgs to both chains
	if clients.Ready() {
		// TODO: Add retry here for out of gas or other errors
		if clients.Send(ctx, src, dst); clients.Success() {
			log.Println(fmt.Sprintf("★ Clients created: [%s]client(%s) and [%s]client(%s)",
				src.ChainID(), src.ClientID(), dst.ChainID(), dst.ClientID()))
		}
	}
	return nil
}
