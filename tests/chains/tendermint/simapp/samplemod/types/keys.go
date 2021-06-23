package types

const (
	// ModuleName defines the module name
	ModuleName = "samplemod"

	// Version defines the current version the samplemod
	// module supports
	Version = "samplemod-1"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// RouterKey is the message route for slashing
	RouterKey = ModuleName

	// QuerierRoute defines the module's query routing key
	QuerierRoute = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_capability"
)
