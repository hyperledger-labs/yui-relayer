package config

import (
	"fmt"

	"github.com/hyperledger-labs/yui-relayer/core"
)

type Chains []*core.ProvableChain

// Get returns the configuration for a given chain
func (cs Chains) Get(chainID string) (*core.ProvableChain, error) {
	for _, chain := range cs {
		if chainID == chain.ChainID() {
			return chain, nil
		}
	}
	return nil, fmt.Errorf("chain with ID %s is not configured", chainID)
}

// Gets returns a map chainIDs to their chains
func (cs Chains) Gets(chainIDs ...string) (map[string]*core.ProvableChain, error) {
	out := make(map[string]*core.ProvableChain)
	for _, cid := range chainIDs {
		chain, err := cs.Get(cid)
		if err != nil {
			return out, err
		}
		out[cid] = chain
	}
	return out, nil
}
