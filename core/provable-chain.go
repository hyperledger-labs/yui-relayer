package core

import (
	"context"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
)

// ProvableChain represents a chain that is supported by the relayer.
type ProvableChain struct {
	Chain
	Prover
}

// NewProvableChain returns a new ProvableChain instance
func NewProvableChain(chain Chain, prover Prover) *ProvableChain {
	return &ProvableChain{Chain: chain, Prover: prover}
}

func (pc *ProvableChain) Init(homePath string, timeout time.Duration, codec codec.ProtoCodecMarshaler, debug bool) error {
	if err := pc.Chain.Init(homePath, timeout, codec, debug); err != nil {
		return err
	}
	if err := pc.Prover.Init(homePath, timeout, codec, debug); err != nil {
		return err
	}
	return nil
}

func (pc *ProvableChain) SetRelayInfo(path *PathEnd, counterparty *ProvableChain, counterpartyPath *PathEnd) error {
	if err := pc.Chain.SetRelayInfo(path, counterparty, counterpartyPath); err != nil {
		return err
	}
	if err := pc.Prover.SetRelayInfo(path, counterparty, counterpartyPath); err != nil {
		return err
	}
	return nil
}

func (pc *ProvableChain) SetupForRelay(ctx context.Context) error {
	if err := pc.Chain.SetupForRelay(ctx); err != nil {
		return err
	}
	if err := pc.Prover.SetupForRelay(ctx); err != nil {
		return err
	}
	return nil
}
