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

	SortTimeoutedPackets(src, dst *ProvableChain, sh SyncHeaders, rp *RelayPackets) (*RelayPackets, errro)

	// RelayPackets executes RecvPacket to the packets contained in `rp` on both chains (`src` and `dst`).
	RelayPackets(src, dst *ProvableChain, rp *RelayPackets, sh SyncHeaders, doExecuteRelaySrc, doExecuteRelayDst bool) (*RelayMsgs, error)

	// UnrelayedAcknowledgements returns packets to execute AcknowledgePacket to on `src` and `dst`.
	// `includeRelayedButUnfinalized` decides if the result includes packets of which acknowledgePacket has been executed but not finalized
	UnrelayedAcknowledgements(src, dst *ProvableChain, sh SyncHeaders, includeRelayedButUnfinalized bool) (*RelayPackets, error)

	// RelayAcknowledgements executes AcknowledgePacket to the packets contained in `rp` on both chains (`src` and `dst`).
	RelayAcknowledgements(src, dst *ProvableChain, rp *RelayPackets, sh SyncHeaders, doExecuteAckSrc, doExecuteAckDst bool) (*RelayMsgs, error)

	// UpdateClients executes UpdateClient only if needed
	UpdateClients(src, dst *ProvableChain, doExecuteRelaySrc, doExecuteRelayDst, doExecuteAckSrc, doExecuteAckDst bool, sh SyncHeaders, doRefresh bool) (*RelayMsgs, error)

	// Send executes submission of msgs to src/dst chains
	Send(src, dst Chain, msgs *RelayMsgs)
}

// StrategyCfg defines which relaying strategy to take for a given path
type StrategyCfg struct {
	Type string `json:"type" yaml:"type"`

	// If set, executions of acknowledgePacket are always skipped on the src chain.
	// Also `UnrelayedAcknowledgements` returns zero packets for the src chain.
	SrcNoack bool `json:"src-noack" yaml:"src-noack"`

	// If set, executions of acknowledgePacket are always skipped on the dst chain
	// Also `UnrelayedAcknowledgements` returns zero packets for the dst chain.
	DstNoack bool `json:"dst-noack" yaml:"dst-noack"`
}

func GetStrategy(cfg StrategyCfg) (StrategyI, error) {
	switch cfg.Type {
	case "naive":
		return NewNaiveStrategy(cfg.SrcNoack, cfg.DstNoack), nil
	default:
		return nil, fmt.Errorf("unknown strategy type '%v'", cfg.Type)
	}
}

// ValidateStrategy validates that the strategy of path `p` is valid
func (p *Path) ValidateStrategy() error {
	switch p.Strategy.Type {
	case (&NaiveStrategy{}).GetType():
		return nil
	default:
		return fmt.Errorf("invalid strategy: %s", p.Strategy.Type)
	}
}
