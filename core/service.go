package core

import (
	"context"
	"time"

	retry "github.com/avast/retry-go"
)

// StartService starts a relay service
func StartService(ctx context.Context, st StrategyI, src, dst *ProvableChain, relayInterval time.Duration) error {
	sh, err := NewSyncHeaders(src, dst)
	if err != nil {
		return err
	}
	srv := NewRelayService(st, src, dst, sh, relayInterval)
	return srv.Start(ctx)
}

type RelayService struct {
	src      *ProvableChain
	dst      *ProvableChain
	st       StrategyI
	sh       SyncHeaders
	interval time.Duration
}

// NewRelayService returns a new service
func NewRelayService(st StrategyI, src, dst *ProvableChain, sh SyncHeaders, interval time.Duration) *RelayService {
	return &RelayService{
		src:      src,
		dst:      dst,
		st:       st,
		sh:       sh,
		interval: interval,
	}
}

// Start starts a relay service
func (srv *RelayService) Start(ctx context.Context) error {
	logger := GetChannelPairLogger(srv.src, srv.dst)
	for {
		if err := retry.Do(func() error {
			select {
			case <-ctx.Done():
				return retry.Unrecoverable(ctx.Err())
			default:
				return srv.Serve(ctx)
			}
		}, rtyAtt, rtyDel, rtyErr, retry.OnRetry(func(n uint, err error) {
			logger.Info(
				"retrying to serve relays",
				"src", srv.src.ChainID(),
				"dst", srv.dst.ChainID(),
				"try", n+1,
				"try_limit", rtyAttNum,
				"error", err.Error(),
			)
		})); err != nil {
			return err
		}
		time.Sleep(srv.interval)
	}
}

// Serve performs packet-relay
func (srv *RelayService) Serve(ctx context.Context) error {
	logger := GetChannelPairLogger(srv.src, srv.dst)
	// First, update the latest headers for src and dst
	if err := srv.sh.Updates(srv.src, srv.dst); err != nil {
		logger.Error("failed to update headers", err)
		return err
	}

	// get unrelayed packets
	pseqs, err := srv.st.UnrelayedPackets(srv.src, srv.dst, srv.sh, false)
	if err != nil {
		logger.Error("failed to get unrelayed sequences", err)
		return err
	}

	// get unrelayed acks
	aseqs, err := srv.st.UnrelayedAcknowledgements(srv.src, srv.dst, srv.sh, false)
	if err != nil {
		logger.Error("failed to get unrelayed acknowledgements", err)
		return err
	}

	// relay packets if unrelayed seqs exist
	if err := srv.st.RelayPackets(srv.src, srv.dst, pseqs, srv.sh); err != nil {
		logger.Error("failed to relay packets", err)
		return err
	}

	// relay acks if unrelayed seqs exist
	if err := srv.st.RelayAcknowledgements(srv.src, srv.dst, aseqs, srv.sh); err != nil {
		logger.Error("failed to relay acknowledgements", err)
		return err
	}

	return nil
}
