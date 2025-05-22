package coreutil

import (
	"fmt"

	"github.com/hyperledger-labs/yui-relayer/core"
	"github.com/hyperledger-labs/yui-relayer/otelcore"
)

// UnwrapChain finds the first struct value in the Chain field that matches the specified
// type argument.
//
// In the following example, UnwrapChain returns a *module.Chain value in the Chain field:
//
//	chain, err := coreutil.UnwrapChain[*module.Chain](provableChain)
func UnwrapChain[C core.Chain](c core.Chain) (C, error) {
	chain := c
	for {
		switch unwrapped := chain.(type) {
		case *core.ProvableChain:
			chain = unwrapped.Chain
		case *otelcore.Chain:
			chain = unwrapped.Chain
		case C:
			return unwrapped, nil
		default:
			var zero C
			return zero, fmt.Errorf("failed to unwrap chain: expected=%T, actual=%T", zero, unwrapped)
		}
	}
}

// UnwrapProver finds the first struct value in the Prover field that matches the specified
// type argument.
//
// In the following example, UnwrapProver returns a *module.Prover value in the Prover field:
//
//	chain, err := coreutil.UnwrapProver[*module.Chain](provableChain)
func UnwrapProver[P core.Prover](p core.Prover) (P, error) {
	prover := p
	for {
		switch unwrapped := prover.(type) {
		case *core.ProvableChain:
			prover = unwrapped.Prover
		case *otelcore.Prover:
			prover = unwrapped.Prover
		case P:
			return unwrapped, nil
		default:
			var zero P
			return zero, fmt.Errorf("failed to unwrap prover: expected=%T, actual=%T", zero, unwrapped)
		}
	}
}
