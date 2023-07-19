package core

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/hyperledger-labs/yui-relayer/logger"
	"golang.org/x/sync/errgroup"
)

func CreateClients(src, dst *ProvableChain) error {
	zapLogger := logger.GetLogger()
	defer zapLogger.Zap.Sync()

	var (
		clients = &RelayMsgs{Src: []sdk.Msg{}, Dst: []sdk.Msg{}}
	)

	srcH, dstH, err := getHeadersForCreateClient(src, dst)
	if err != nil {
		clientErrorwChannel(
			zapLogger,
			"failed to get headers for create client",
			src, dst,
			err,
		)
		return err
	}

	srcAddr, err := src.GetAddress()
	if err != nil {
		clientErrorwChannel(
			zapLogger,
			"failed to get address for create client",
			src, dst,
			err,
		)
		return err
	}
	dstAddr, err := dst.GetAddress()
	if err != nil {
		clientErrorwChannel(
			zapLogger,
			"failed to get address for create client",
			src, dst,
			err,
		)
		return err
	}

	{
		msg, err := dst.CreateMsgCreateClient(src.Path().ClientID, dstH, srcAddr)
		if err != nil {
			clientErrorwChannel(
				zapLogger,
				"failed to create client",
				src, dst,
				err,
			)
			return err
		}
		clients.Src = append(clients.Src, msg)
	}

	{
		msg, err := src.CreateMsgCreateClient(dst.Path().ClientID, srcH, dstAddr)
		if err != nil {
			clientErrorwChannel(
				zapLogger,
				"failed to create client",
				src, dst,
				err,
			)
			return err
		}
		clients.Dst = append(clients.Dst, msg)
	}

	// Send msgs to both chains
	if clients.Ready() {
		// TODO: Add retry here for out of gas or other errors
		if clients.Send(src, dst); clients.Success() {
			clientInfowChannel(
				zapLogger,
				"★ Clients created",
				src, dst,
			)
		}
	}
	return nil
}

func UpdateClients(src, dst *ProvableChain) error {
	zapLogger := logger.GetLogger()
	defer zapLogger.Zap.Sync()
	var (
		clients = &RelayMsgs{Src: []sdk.Msg{}, Dst: []sdk.Msg{}}
	)
	// First, update the light clients to the latest header and return the header
	sh, err := NewSyncHeaders(src, dst)
	if err != nil {
		clientErrorwChannel(
			zapLogger,
			"failed to create sync headers for update client",
			src, dst,
			err,
		)
		return err
	}
	srcUpdateHeaders, dstUpdateHeaders, err := sh.SetupBothHeadersForUpdate(src, dst)
	if err != nil {
		clientErrorwChannel(
			zapLogger,
			"failed to setup both headers for update client",
			src, dst,
			err,
		)
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
			clientInfowChannel(
				zapLogger,
				"★ Clients updated",
				src, dst,
			)
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

func clientErrorwChannel(zapLogger *logger.ZapLogger, msg string, src, dst *ProvableChain, err error) {
	zapLogger.ErrorwChannel(
		msg,
		src.ChainID(), src.Path().ChannelID, src.Path().PortID,
		dst.ChainID(), dst.Path().ChannelID, dst.Path().PortID,
		err,
		"core.client",
	)
}

func clientInfowChannel(zapLogger *logger.ZapLogger, msg string, src, dst *ProvableChain) {
	zapLogger.InfowChannel(
		msg,
		src.ChainID(), src.Path().ChannelID, src.Path().PortID,
		dst.ChainID(), dst.Path().ChannelID, dst.Path().PortID,
		"",
	)
}
