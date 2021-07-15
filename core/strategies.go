package core

import (
	"fmt"
)

// StrategyI defines
type StrategyI interface {
	GetType() string
	UnrelayedSequences(src, dst *ProvableChain, sh SyncHeadersI) (*RelaySequences, error)
	RelayPackets(src, dst *ProvableChain, sp *RelaySequences, sh SyncHeadersI) error
	UnrelayedAcknowledgements(src, dst *ProvableChain, sh SyncHeadersI) (*RelaySequences, error)
	RelayAcknowledgements(src, dst *ProvableChain, sp *RelaySequences, sh SyncHeadersI) error
}

// StrategyCfg defines which relaying strategy to take for a given path
type StrategyCfg struct {
	Type string `json:"type" yaml:"type"`
}

func GetStrategy(cfg StrategyCfg) (StrategyI, error) {
	switch cfg.Type {
	case "naive":
		return NewNaiveStrategy(), nil
	default:
		return nil, fmt.Errorf("unknown strategy type '%v'", cfg.Type)
	}
}

// RunStrategy runs a given strategy
func RunStrategy(src, dst *ProvableChain, strategy StrategyI) (func(), error) {
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

// GetStrategy the strategy defined in the relay messages
func (p *Path) GetStrategy() (StrategyI, error) {
	switch p.Strategy.Type {
	case (&NaiveStrategy{}).GetType():
		return &NaiveStrategy{}, nil
	default:
		return nil, fmt.Errorf("invalid strategy: %s", p.Strategy.Type)
	}
}
