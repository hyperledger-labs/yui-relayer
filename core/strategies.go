package core

import (
	"context"
	"fmt"
)

// StrategyI defines
type StrategyI interface {
	GetType() string
	SetupRelay(ctx context.Context, src, dst *ProvableChain) error
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

// GetStrategy the strategy defined in the relay messages
func (p *Path) GetStrategy() (StrategyI, error) {
	switch p.Strategy.Type {
	case (&NaiveStrategy{}).GetType():
		return &NaiveStrategy{}, nil
	default:
		return nil, fmt.Errorf("invalid strategy: %s", p.Strategy.Type)
	}
}
