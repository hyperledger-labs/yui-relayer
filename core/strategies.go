package core

import (
	"context"
	"fmt"
)

// StrategyI defines
type StrategyI interface {
	// GetType returns the type of the strategy
	GetType() string

	// SetupRelay performs chain-specific setup for `src` and `dst` before starting the relay
	SetupRelay(ctx context.Context, src, dst *ProvableChain) error

	// UnrelayedPackets returns packets to execute RecvPacket to on `src` and `dst`.
	// `includeRelayedButUnfinalized` decides if the result includes packets of which recvPacket has been executed but not finalized
	UnrelayedPackets(src, dst *ProvableChain, sh SyncHeaders, includeRelayedButUnfinalized bool) (*RelayPackets, error)

	// RelayPackets executes RecvPacket to the packets contained in `rp` on both chains (`src` and `dst`).
	RelayPackets(src, dst *ProvableChain, rp *RelayPackets, sh SyncHeaders) error

	// UnrelayedAcknowledgements returns packets to execute AcknowledgePacket to on `src` and `dst`.
	// `includeRelayedButUnfinalized` decides if the result includes packets of which acknowledgePacket has been executed but not finalized
	UnrelayedAcknowledgements(src, dst *ProvableChain, sh SyncHeaders, includeRelayedButUnfinalized bool) (*RelayPackets, error)

	// RelayAcknowledgements executes AcknowledgePacket to the packets contained in `rp` on both chains (`src` and `dst`).
	RelayAcknowledgements(src, dst *ProvableChain, ra *RelayPackets, sh SyncHeaders) error
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
