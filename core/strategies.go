package core

import (
	"fmt"

	"github.com/cosmos/relayer/relayer"
)

// StrategyI defines
type StrategyI interface {
	GetType() string
	UnrelayedSequences(src, dst ChainI, sh SyncHeadersI) (*RelaySequences, error)
	RelayPackets(src, dst ChainI, sp *RelaySequences, sh SyncHeadersI) error
	UnrelayedAcknowledgements(src, dst ChainI, sh SyncHeadersI) (*RelaySequences, error)
	// RelayAcknowledgements(src, dst ChainI, sp *RelaySequences, sh SyncHeadersI) error
}

func GetStrategy(cfg relayer.StrategyCfg) (StrategyI, error) {
	switch cfg.Type {
	case "naive":
		return NewNaiveStrategy(), nil
	default:
		return nil, fmt.Errorf("unknown strategy type '%v'", cfg.Type)
	}
}

// RunStrategy runs a given strategy
func RunStrategy(src, dst ChainI, strategy StrategyI) (func(), error) {
	doneChan := make(chan struct{})

	// Fetch latest headers for each chain and store them in sync headers
	sh, err := NewSyncHeaders(src, dst)
	if err != nil {
		return nil, err
	}

	// Next start the goroutine that listens to each chain for block and tx events
	go src.StartEventListener(dst, strategy)
	go dst.StartEventListener(src, strategy)

	// Fetch any unrelayed sequences depending on the channel order
	sp, err := strategy.UnrelayedSequences(src, dst, sh)
	if err != nil {
		return nil, err
	}

	if err = strategy.RelayPackets(src, dst, sp, sh); err != nil {
		return nil, err
	}

	// Return a function to stop the relayer goroutine
	return func() { doneChan <- struct{}{} }, nil
}
