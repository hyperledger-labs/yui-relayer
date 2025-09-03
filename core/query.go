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
			err = ProveClientState(srcCtx, src, srcCsRes)
			if err != nil {
				return err
			}
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
			err = ProveClientState(dstCtx, dst, dstCsRes)
			if err != nil {
				return err
			}
		}
		return err
	})
	err = eg.Wait()
	return
}

func ProveClientState(
	ctx QueryContext,
	chain interface {
		Chain
		StateProver
	},
	csRes *clienttypes.QueryClientStateResponse,
) error {
	path := host.FullClientStatePath(chain.Path().ClientID)

	value, err := chain.Codec().Marshal(csRes.ClientState)
	if err != nil {
		return err
	}
	csRes.Proof, csRes.ProofHeight, err = chain.ProveState(ctx, path, value)
	if err != nil {
		return err
	}
	return nil
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
			err = ProveClientConsensusState(srcCtx, src, srcClientConsH, srcCsRes)
			if err != nil {
				return err
			}
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
			err = ProveClientConsensusState(dstCtx, dst, dstClientConsH, dstCsRes)
			if err != nil {
				return err
			}
		}
		return err
	})
	err = eg.Wait()
	return
}

func ProveClientConsensusState(
	ctx QueryContext,
	chain interface {
		Chain
		StateProver
	},
	clientConsH ibcexported.Height,
	csRes *clienttypes.QueryConsensusStateResponse,
) error {
	path := host.FullConsensusStatePath(chain.Path().ClientID, clientConsH)
	value, err := chain.Codec().Marshal(csRes.ConsensusState)
	if err != nil {
		return err
	}
	csRes.Proof, csRes.ProofHeight, err = chain.ProveState(ctx, path, value)
	if err != nil {
		return err
	}
	return nil
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
			err := ProveConnection(srcCtx, src, srcConn)
			if err != nil {
				return err
			}
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
			err := ProveConnection(dstCtx, dst, dstConn)
			if err != nil {
				return err
			}
		}
		return err
	})
	err = eg.Wait()
	return
}

func ProveConnection(
	queryCtx QueryContext,
	chain interface {
		Chain
		StateProver
	},
	conn *conntypes.QueryConnectionResponse,
) error {
	if conn.Connection.State == conntypes.UNINITIALIZED {
		return nil
	}

	path := host.ConnectionPath(chain.Path().ConnectionID)
	value, err := chain.Codec().Marshal(conn.Connection)
	if err != nil {
		return err
	}
	conn.Proof, conn.ProofHeight, err = chain.ProveState(queryCtx, path, value)
	return err
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
			err := ProveChannel(srcCtx, src, srcChan)
			if err != nil {
				return err
			}
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
			err := ProveChannel(dstCtx, dst, dstChan)
			if err != nil {
				return err
			}
		}
		return err
	})
	err = eg.Wait()
	return
}

func ProveChannel(
	queryCtx QueryContext,
	chain interface {
		Chain
		StateProver
	},
	ch *chantypes.QueryChannelResponse,
) error {
	if ch.Channel.State == chantypes.UNINITIALIZED {
		return nil
	}

	path := host.ChannelPath(chain.Path().PortID, chain.Path().ChannelID)
	value, err := chain.Codec().Marshal(ch.Channel)
	if err != nil {
		return err
	}
	ch.Proof, ch.ProofHeight, err = chain.ProveState(queryCtx, path, value)
	if err != nil {
		return err
	}
	return nil
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

		if err := ProveChannelUpgrade(srcCtx, src, srcChanUpg); err != nil {
			return err
		}
		return nil
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

		if err := ProveChannelUpgrade(dstCtx, dst, dstChanUpg); err != nil {
			return err
		}
		return nil
	})
	err = eg.Wait()
	return
}

func ProveChannelUpgrade(ctx QueryContext, ch interface {
	Chain
	StateProver
}, chanUpg *chantypes.QueryUpgradeResponse) error {
	value, err := ch.Codec().Marshal(&chanUpg.Upgrade)
	if err != nil {
		return err
	}
	path := host.ChannelUpgradePath(ch.Path().PortID, ch.Path().ChannelID)
	chanUpg.Proof, chanUpg.ProofHeight, err = ch.ProveState(ctx, path, value)
	return err
}

func QueryChannelUpgradeError(ctx QueryContext, chain interface {
	Chain
	StateProver
}, prove bool) (*chantypes.QueryUpgradeErrorResponse, error) {
	if chanUpgErr, err := chain.QueryChannelUpgradeError(ctx); err != nil {
		return nil, err
	} else if chanUpgErr == nil {
		return nil, nil
	} else if !prove {
		return chanUpgErr, nil
	} else if value, err := chain.Codec().Marshal(&chanUpgErr.ErrorReceipt); err != nil {
		return nil, err
	} else {
		path := host.ChannelUpgradeErrorPath(chain.Path().PortID, chain.Path().ChannelID)
		chanUpgErr.Proof, chanUpgErr.ProofHeight, err = chain.ProveState(ctx, path, value)
		return chanUpgErr, err
	}
}
