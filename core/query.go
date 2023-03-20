package core

import (
	clienttypes "github.com/cosmos/ibc-go/v4/modules/core/02-client/types"
	conntypes "github.com/cosmos/ibc-go/v4/modules/core/03-connection/types"
	chantypes "github.com/cosmos/ibc-go/v4/modules/core/04-channel/types"
	ibcexported "github.com/cosmos/ibc-go/v4/modules/core/exported"
	"golang.org/x/sync/errgroup"
)

// QueryClientStatePair returns a pair of connection responses
func QueryClientStatePair(
	srcCtx, dstCtx QueryContext,
	src, dst IBCProvableQuerier,
) (srcCsRes, dstCsRes *clienttypes.QueryClientStateResponse, err error) {
	var eg = new(errgroup.Group)
	eg.Go(func() error {
		srcCsRes, err = src.QueryClientStateWithProof(srcCtx)
		return err
	})
	eg.Go(func() error {
		dstCsRes, err = dst.QueryClientStateWithProof(dstCtx)
		return err
	})
	err = eg.Wait()
	return
}

// QueryClientConsensusStatePair allows for the querying of multiple client states at the same time
func QueryClientConsensusStatePair(
	srcCtx, dstCtx QueryContext,
	src, dst IBCProvableQuerier,
	srcClientConsH,
	dstClientConsH ibcexported.Height) (srcCsRes, dstCsRes *clienttypes.QueryConsensusStateResponse, err error) {
	var eg = new(errgroup.Group)
	eg.Go(func() error {
		srcCsRes, err = src.QueryClientConsensusStateWithProof(srcCtx, srcClientConsH)
		return err
	})
	eg.Go(func() error {
		dstCsRes, err = dst.QueryClientConsensusStateWithProof(dstCtx, dstClientConsH)
		return err
	})
	err = eg.Wait()
	return
}

// QueryConnectionPair returns a pair of connection responses
func QueryConnectionPair(
	srcCtx, dstCtx QueryContext,
	src, dst IBCProvableQuerier,
) (srcConn, dstConn *conntypes.QueryConnectionResponse, err error) {
	var eg = new(errgroup.Group)
	eg.Go(func() error {
		srcConn, err = src.QueryConnectionWithProof(srcCtx)
		return err
	})
	eg.Go(func() error {
		dstConn, err = dst.QueryConnectionWithProof(dstCtx)
		return err
	})
	err = eg.Wait()
	return
}

// QueryChannelPair returns a pair of channel responses
func QueryChannelPair(srcCtx, dstCtx QueryContext, src, dst IBCProvableQuerier) (srcChan, dstChan *chantypes.QueryChannelResponse, err error) {
	var eg = new(errgroup.Group)
	eg.Go(func() error {
		srcChan, err = src.QueryChannelWithProof(srcCtx)
		return err
	})
	eg.Go(func() error {
		dstChan, err = dst.QueryChannelWithProof(dstCtx)
		return err
	})
	err = eg.Wait()
	return
}
