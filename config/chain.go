package config

import (
	"fmt"

	"github.com/datachainlab/relayer/core"
)

type Chains []core.ChainI

// Get returns the configuration for a given chain
func (c Chains) Get(chainID string) (core.ChainI, error) {
	for _, chain := range c {
		if chainID == chain.ChainID() {
			return chain, nil
		}
	}
	return nil, fmt.Errorf("chain with ID %s is not configured", chainID)
}

// Gets returns a map chainIDs to their chains
func (c Chains) Gets(chainIDs ...string) (map[string]core.ChainI, error) {
	out := make(map[string]core.ChainI)
	for _, cid := range chainIDs {
		chain, err := c.Get(cid)
		if err != nil {
			return out, err
		}
		out[cid] = chain
	}
	return out, nil
}
