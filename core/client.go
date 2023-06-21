package core

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/hyperledger-labs/yui-relayer/logger"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

func CreateClients(src, dst *ProvableChain) error {
	logger := logger.ZapLogger()
	defer logger.Sync()
	var (
		clients = &RelayMsgs{Src: []sdk.Msg{}, Dst: []sdk.Msg{}}
	)

	srcH, dstH, err := getHeadersForCreateClient(src, dst)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to get headers for create client [%s]chan{%s}port{%s} -> [%s]chan{%s}port{%s}",
			src.ChainID(), src.Path().ChannelID, src.Path().PortID,
			dst.ChainID(), dst.Path().ChannelID, dst.Path().PortID),
			zap.Error(err))
		return err
	}

	srcAddr, err := src.GetAddress()
	if err != nil {
		logger.Error(fmt.Sprintf("failed to get address for create client [src: %s]", src.ChainID()), zap.Error(err))
		return err
	}
	dstAddr, err := dst.GetAddress()
	if err != nil {
		logger.Error(fmt.Sprintf("failed to get address for create client [dst: %s]", dst.ChainID()), zap.Error(err))
		return err
	}

	{
		msg, err := dst.CreateMsgCreateClient(src.Path().ClientID, dstH, srcAddr)
		if err != nil {
			logger.Error(fmt.Sprintf("failed to create client on dst chain [%s]chan{%s}port{%s} -> [%s]chan{%s}port{%s}",
				src.ChainID(), src.Path().ChannelID, src.Path().PortID,
				dst.ChainID(), dst.Path().ChannelID, dst.Path().PortID),
				zap.Error(err))
			return err
		}
		clients.Src = append(clients.Src, msg)
	}

	{
		msg, err := src.CreateMsgCreateClient(dst.Path().ClientID, srcH, dstAddr)
		if err != nil {
			logger.Error(fmt.Sprintf("failed to create client on src chain [%s]chan{%s}port{%s} -> [%s]chan{%s}port{%s}",
				src.ChainID(), src.Path().ChannelID, src.Path().PortID,
				dst.ChainID(), dst.Path().ChannelID, dst.Path().PortID),
				zap.Error(err))
			return err
		}
		clients.Dst = append(clients.Dst, msg)
	}

	// Send msgs to both chains
	if clients.Ready() {
		// TODO: Add retry here for out of gas or other errors
		if clients.Send(src, dst); clients.Success() {
			logger.Info(fmt.Sprintf("★ Clients created: [%s]client(%s) and [%s]client(%s)",
				src.ChainID(), src.Path().ClientID, dst.ChainID(), dst.Path().ClientID))
		}
	}
	return nil
}

func UpdateClients(src, dst *ProvableChain) error {
	logger := logger.ZapLogger()
	defer logger.Sync()
	var (
		clients = &RelayMsgs{Src: []sdk.Msg{}, Dst: []sdk.Msg{}}
	)
	// First, update the light clients to the latest header and return the header
	sh, err := NewSyncHeaders(src, dst)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to create sync headers for update client [%s]chan{%s}port{%s} -> [%s]chan{%s}port{%s}",
			src.ChainID(), src.Path().ChannelID, src.Path().PortID,
			dst.ChainID(), dst.Path().ChannelID, dst.Path().PortID),
			zap.Error(err))
		return err
	}
	srcUpdateHeaders, dstUpdateHeaders, err := sh.SetupBothHeadersForUpdate(src, dst)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to setup both headers for update client [%s]chan{%s}port{%s} -> [%s]chan{%s}port{%s}",
			src.ChainID(), src.Path().ChannelID, src.Path().PortID,
			dst.ChainID(), dst.Path().ChannelID, dst.Path().PortID),
			zap.Error(err))
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
			logger.Info(fmt.Sprintf("★ Clients updated: [%s]client(%s) and [%s]client(%s)",
				src.ChainID(), src.Path().ClientID, dst.ChainID(), dst.Path().ClientID))
		}
	}
	return nil
}

// getHeadersForCreateClient calls UpdateLightWithHeader on the passed chains concurrently
func getHeadersForCreateClient(src, dst LightClient) (srch, dsth Header, err error) {
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
