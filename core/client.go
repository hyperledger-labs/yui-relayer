package core

import (
	"log"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"golang.org/x/sync/errgroup"
)

func CreateClients(src, dst *ProvableChain) error {
	var (
		clients = &RelayMsgs{Src: []sdk.Msg{}, Dst: []sdk.Msg{}}
	)

	srcH, dstH, err := getHeadersForCreateClient(src, dst)
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
		msg, err := dst.CreateMsgCreateClient(src.Path().ClientID, dstH, srcAddr)
		if err != nil {
			return err
		}
		clients.Src = append(clients.Src, msg)
	}

	{
		msg, err := src.CreateMsgCreateClient(dst.Path().ClientID, srcH, dstAddr)
		if err != nil {
			return err
		}
		clients.Dst = append(clients.Dst, msg)
	}

	// Send msgs to both chains
	if clients.Ready() {
		// TODO: Add retry here for out of gas or other errors
		if clients.Send(src, dst); clients.Success() {
			log.Printf("★ Clients created: [%s]client(%s) and [%s]client(%s)",
				src.ChainID(), src.Path().ClientID, dst.ChainID(), dst.Path().ClientID)
		}
	}
	return nil
}

func UpdateClients(src, dst *ProvableChain) error {
	var (
		clients = &RelayMsgs{Src: []sdk.Msg{}, Dst: []sdk.Msg{}}
	)
	// First, update the light clients to the latest header and return the header
	sh, err := NewSyncHeaders(src, dst)
	if err != nil {
		return err
	}
	srcUpdateHeaders, dstUpdateHeaders, err := sh.SetupBothHeadersForUpdate(src, dst)
	if err != nil {
		return err
	}
	if len(dstUpdateHeaders) > 0 {
		clients.Src = append(clients.Src, src.Path().UpdateClients(dstUpdateHeaders, mustGetAddress(src))...)
	}
	if len(srcUpdateHeaders) > 0 {
		clients.Dst = append(clients.Dst, dst.Path().UpdateClients(srcUpdateHeaders, mustGetAddress(dst))...)
	}
	// Send msgs to both chains
	if clients.Ready() {
		if clients.Send(src, dst); clients.Success() {
			log.Printf("★ Clients updated: [%s]client(%s) and [%s]client(%s)",
				src.ChainID(), src.Path().ClientID, dst.ChainID(), dst.Path().ClientID)
		}
	}
	return nil
}

// getHeadersForCreateClient calls UpdateLightWithHeader on the passed chains concurrently
func getHeadersForCreateClient(src, dst LightClient) (srch, dsth HeaderI, err error) {
	var eg = new(errgroup.Group)
	eg.Go(func() error {
		srch, err = src.GetLatestFinalizedHeader()
		return err
	})
	eg.Go(func() error {
		dsth, err = dst.GetLatestFinalizedHeader()
		return err
	})
	if err := eg.Wait(); err != nil {
		return nil, nil, err
	}
	return srch, dsth, nil
}
