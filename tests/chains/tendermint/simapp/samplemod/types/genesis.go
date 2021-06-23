package types

// NewGenesisState creates a new cross module GenesisState instance.
func NewGenesisState() *GenesisState {
	return &GenesisState{}
}

// DefaultGenesis returns a GenesisState instance
func DefaultGenesis() *GenesisState {
	return &GenesisState{}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	return nil
}
