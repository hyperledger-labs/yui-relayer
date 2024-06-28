package core

import (
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	conntypes "github.com/cosmos/ibc-go/v8/modules/core/03-connection/types"
	chantypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
	ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"
	"golang.org/x/sync/errgroup"
)

// QueryClientStatePair returns a pair of connection responses
func QueryClientStatePair(
	srcCtx, dstCtx QueryContext,
	src, dst interface {
		Chain
		StateProver
	},
	prove bool,
) (srcCsRes, dstCsRes *clienttypes.QueryClientStateResponse, err error) {
	var eg = new(errgroup.Group)
	eg.Go(func() error {
		var err error
		srcCsRes, err = src.QueryClientState(srcCtx)
		if err != nil {
			return err
		}
		if prove {
			path := host.FullClientStatePath(src.Path().ClientID)
			var value []byte
			value, err = src.Codec().Marshal(srcCsRes.ClientState)
			if err != nil {
				return err
			}
			srcCsRes.Proof, srcCsRes.ProofHeight, err = src.ProveState(srcCtx, path, value)
		}
		return err
	})
	eg.Go(func() error {
		var err error
		dstCsRes, err = dst.QueryClientState(dstCtx)
		if err != nil {
			return err
		}
		if prove {
			path := host.FullClientStatePath(dst.Path().ClientID)
			var value []byte
			value, err = dst.Codec().Marshal(dstCsRes.ClientState)
			if err != nil {
				return err
			}
			dstCsRes.Proof, dstCsRes.ProofHeight, err = dst.ProveState(dstCtx, path, value)
		}
		return err
	})
	err = eg.Wait()
	return
}

// QueryClientConsensusStatePair allows for the querying of multiple client states at the same time
func QueryClientConsensusStatePair(
	srcCtx, dstCtx QueryContext,
	src, dst interface {
		Chain
		StateProver
	},
	srcClientConsH,
	dstClientConsH ibcexported.Height,
	prove bool,
) (srcCsRes, dstCsRes *clienttypes.QueryConsensusStateResponse, err error) {
	var eg = new(errgroup.Group)
	eg.Go(func() error {
		var err error
		srcCsRes, err = src.QueryClientConsensusState(srcCtx, srcClientConsH)
		if err != nil {
			return err
		}
		if prove {
			path := host.FullConsensusStatePath(src.Path().ClientID, srcClientConsH)
			var value []byte
			value, err = src.Codec().Marshal(srcCsRes.ConsensusState)
			if err != nil {
				return err
			}
			srcCsRes.Proof, srcCsRes.ProofHeight, err = src.ProveState(srcCtx, path, value)
		}
		return err
	})
	eg.Go(func() error {
		var err error
		dstCsRes, err = dst.QueryClientConsensusState(dstCtx, dstClientConsH)
		if err != nil {
			return err
		}
		if prove {
			path := host.FullConsensusStatePath(dst.Path().ClientID, dstClientConsH)
			var value []byte
			value, err = dst.Codec().Marshal(dstCsRes.ConsensusState)
			if err != nil {
				return err
			}
			dstCsRes.Proof, dstCsRes.ProofHeight, err = dst.ProveState(dstCtx, path, value)
		}
		return err
	})
	err = eg.Wait()
	return
}

// QueryConnectionPair returns a pair of connection responses
func QueryConnectionPair(
	srcCtx, dstCtx QueryContext,
	src, dst interface {
		Chain
		StateProver
	},
	prove bool,
) (srcConn, dstConn *conntypes.QueryConnectionResponse, err error) {
	var eg = new(errgroup.Group)
	eg.Go(func() error {
		if src.Path().ConnectionID == "" {
			srcConn = &conntypes.QueryConnectionResponse{
				Connection: &conntypes.ConnectionEnd{
					State: conntypes.UNINITIALIZED,
				},
			}
			return nil
		}
		var err error
		srcConn, err = src.QueryConnection(srcCtx, src.Path().ConnectionID)
		if err != nil {
			return err
		} else if srcConn.Connection.State == conntypes.UNINITIALIZED {
			return nil
		}
		if prove {
			path := host.ConnectionPath(src.Path().ConnectionID)
			var value []byte
			value, err = src.Codec().Marshal(srcConn.Connection)
			if err != nil {
				return err
			}
			srcConn.Proof, srcConn.ProofHeight, err = src.ProveState(srcCtx, path, value)
		}
		return err
	})
	eg.Go(func() error {
		if dst.Path().ConnectionID == "" {
			dstConn = &conntypes.QueryConnectionResponse{
				Connection: &conntypes.ConnectionEnd{
					State: conntypes.UNINITIALIZED,
				},
			}
			return nil
		}
		var err error
		dstConn, err = dst.QueryConnection(dstCtx, dst.Path().ConnectionID)
		if err != nil {
			return err
		} else if dstConn.Connection.State == conntypes.UNINITIALIZED {
			return nil
		}
		if prove {
			path := host.ConnectionPath(dst.Path().ConnectionID)
			var value []byte
			value, err = dst.Codec().Marshal(dstConn.Connection)
			if err != nil {
				return err
			}
			dstConn.Proof, dstConn.ProofHeight, err = dst.ProveState(dstCtx, path, value)
		}
		return err
	})
	err = eg.Wait()
	return
}

// QueryChannelPair returns a pair of channel responses
func QueryChannelPair(srcCtx, dstCtx QueryContext, src, dst interface {
	Chain
	StateProver
}, prove bool) (srcChan, dstChan *chantypes.QueryChannelResponse, err error) {
	var eg = new(errgroup.Group)
	eg.Go(func() error {
		if src.Path().ChannelID == "" {
			srcChan = &chantypes.QueryChannelResponse{
				Channel: &chantypes.Channel{
					State: chantypes.UNINITIALIZED,
				},
			}
			return nil
		}
		var err error
		srcChan, err = src.QueryChannel(srcCtx)
		if err != nil {
			return err
		} else if srcChan.Channel.State == chantypes.UNINITIALIZED {
			return nil
		}
		if prove {
			path := host.ChannelPath(src.Path().PortID, src.Path().ChannelID)
			var value []byte
			value, err = src.Codec().Marshal(srcChan.Channel)
			if err != nil {
				return err
			}
			srcChan.Proof, srcChan.ProofHeight, err = src.ProveState(srcCtx, path, value)
		}
		return err
	})
	eg.Go(func() error {
		if dst.Path().ChannelID == "" {
			dstChan = &chantypes.QueryChannelResponse{
				Channel: &chantypes.Channel{
					State: chantypes.UNINITIALIZED,
				},
			}
			return nil
		}
		var err error
		dstChan, err = dst.QueryChannel(dstCtx)
		if err != nil {
			return err
		} else if dstChan.Channel.State == chantypes.UNINITIALIZED {
			return nil
		}
		if prove {
			path := host.ChannelPath(dst.Path().PortID, dst.Path().ChannelID)
			var value []byte
			value, err = dst.Codec().Marshal(dstChan.Channel)
			if err != nil {
				return err
			}
			dstChan.Proof, dstChan.ProofHeight, err = dst.ProveState(dstCtx, path, value)
		}
		return err
	})
	err = eg.Wait()
	return
}

func QueryChannelUpgradePair(srcCtx, dstCtx QueryContext, src, dst interface {
	Chain
	StateProver
}, prove bool) (srcChanUpg, dstChanUpg *chantypes.QueryUpgradeResponse, err error) {
	eg := new(errgroup.Group)

	// get channel upgrade from src chain
	eg.Go(func() error {
		var err error
		srcChanUpg, err = src.QueryChannelUpgrade(srcCtx)
		if err != nil {
			return err
		} else if srcChanUpg == nil {
			return nil
		}

		if !prove {
			return nil
		}

		if value, err := src.Codec().Marshal(&srcChanUpg.Upgrade); err != nil {
			return err
		} else {
			path := host.ChannelUpgradePath(src.Path().PortID, src.Path().ChannelID)
			srcChanUpg.Proof, srcChanUpg.ProofHeight, err = src.ProveState(srcCtx, path, value)
			return err
		}
	})

	// get channel upgrade from dst chain
	eg.Go(func() error {
		var err error
		dstChanUpg, err = dst.QueryChannelUpgrade(dstCtx)
		if err != nil {
			return err
		} else if dstChanUpg == nil {
			return nil
		}

		if !prove {
			return nil
		}

		if value, err := dst.Codec().Marshal(&dstChanUpg.Upgrade); err != nil {
			return err
		} else {
			path := host.ChannelUpgradePath(dst.Path().PortID, dst.Path().ChannelID)
			dstChanUpg.Proof, dstChanUpg.ProofHeight, err = dst.ProveState(dstCtx, path, value)
			return err
		}
	})
	err = eg.Wait()
	return
}

func QueryChannelUpgradeErrorPair(srcCtx, dstCtx QueryContext, src, dst interface {
	Chain
	StateProver
}, prove bool) (srcChanUpgErr, dstChanUpgErr *chantypes.QueryUpgradeErrorResponse, err error) {
	eg := new(errgroup.Group)

	// get channel upgrade from src chain
	eg.Go(func() error {
		var err error
		srcChanUpgErr, err = src.QueryChannelUpgradeError(srcCtx)
		if err != nil {
			return err
		} else if srcChanUpgErr == nil {
			return nil
		}

		if !prove {
			return nil
		}

		if value, err := src.Codec().Marshal(&srcChanUpgErr.ErrorReceipt); err != nil {
			return err
		} else {
			path := host.ChannelUpgradeErrorPath(src.Path().PortID, src.Path().ChannelID)
			srcChanUpgErr.Proof, srcChanUpgErr.ProofHeight, err = src.ProveState(srcCtx, path, value)
			return err
		}
	})

	// get channel upgrade from dst chain
	eg.Go(func() error {
		var err error
		dstChanUpgErr, err = dst.QueryChannelUpgradeError(dstCtx)
		if err != nil {
			return err
		} else if dstChanUpgErr == nil {
			return nil
		}

		if !prove {
			return nil
		}

		if value, err := dst.Codec().Marshal(&dstChanUpgErr.ErrorReceipt); err != nil {
			return err
		} else {
			path := host.ChannelUpgradeErrorPath(dst.Path().PortID, dst.Path().ChannelID)
			dstChanUpgErr.Proof, dstChanUpgErr.ProofHeight, err = dst.ProveState(dstCtx, path, value)
			return err
		}
	})
	err = eg.Wait()
	return
}
