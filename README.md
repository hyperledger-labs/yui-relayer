# Relayer

![Test](https://github.com/hyperledger-labs/yui-relayer/workflows/Test/badge.svg)
[![GoDoc](https://godoc.org/github.com/hyperledger-labs/yui-relayer?status.svg)](https://pkg.go.dev/github.com/hyperledger-labs/yui-relayer?tab=doc)

An [IBC](https://github.com/cosmos/ibc) [relayer](https://github.com/cosmos/ibc/tree/main/spec/relayer/ics-018-relayer-algorithms) implementation supports heterogeneous blockchains.

## Supported chains

- Cosmos/Tendermint(with [ibc-go](https://github.com/cosmos/ibc-go))
  - This implementation is a fork of [cosmos/relayer](https://github.com/cosmos/relayer)
- EVM chains(with [ibc-solidity](https://github.com/hyperledger-labs/yui-ibc-solidity))
- Hyperledger Fabric(with [fabric-ibc](https://github.com/hyperledger-labs/yui-fabric-ibc))
- Corda(with [corda-ibc](https://github.com/hyperledger-labs/yui-corda-ibc))

You can find a list of each supported combination of chains and examples of E2E testing with the Relayer here: https://github.com/datachainlab/yui-relayer-build

## Compatibility with IBC

The Relayer uses "vX.Y.Z" as its version format. "v0.Y.Z" will be used until the Relayer's core and module interface is stable.

In addition, "Y" corresponds to the specific major version of ibc-go (i.e., "X"). The following table shows the Relayer version and its corresponding ibc-go version.

| Relayer                                                                       | ibc-go       |
|-------------------------------------------------------------------------------|--------------|
| v0.4.0(current branch)                                                        | v7.0.0       |
| [v0.3.0](https://github.com/hyperledger-labs/yui-relayer/releases/tag/v0.3.0) | v4.0.0       |
| [v0.2.0](https://github.com/hyperledger-labs/yui-relayer/releases/tag/v0.2.0) | v1.0.0-beta1 |

## Glossary

- **Chain**: supports sending a transaction to the chain and querying its state
- **Prover**: generates or query a proof of a target chain's state. This proof is verified by on-chain Light Client deployed on the counterparty chain.
- **ProvableChain**: consists of a Chain and a Prover.
- **Path**: is a path of two ProvableChains that relay packets to each other.
- **ChainConfig**: is a configuration to generate a Chain. It requires implementing `Build` method to build the Chain.
- **ProverConfig**: is a configuration to generate a Prover. It also requires implementing `Build` method to build the Prover.

## How to support a new chain

The Relayer can support additional chains you want without forking the repository.

You can use the Relayer as a library to configure your relayer that supports any chain or Light Client. You must provide a module that implements [Module interface](./config/module.go). 

The following is the implementation of the tendermint module:

```go
import (
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/hyperledger-labs/yui-relayer/chains/tendermint"
	"github.com/hyperledger-labs/yui-relayer/chains/tendermint/cmd"
	"github.com/hyperledger-labs/yui-relayer/config"
	"github.com/spf13/cobra"
)

type Module struct{}

var _ config.ModuleI = (*Module)(nil)

// Name returns the name of the module
func (Module) Name() string {
	return "tendermint"
}

// RegisterInterfaces register the module interfaces to protobuf Any.
func (Module) RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations(
		(*core.ChainConfig)(nil),
		&ChainConfig{},
	)
	registry.RegisterImplementations(
		(*core.ProverConfig)(nil),
		&ProverConfig{},
	)
}

// GetCmd returns the command
func (Module) GetCmd(ctx *config.Context) *cobra.Command {
	return cmd.TendermintCmd(ctx.Codec, ctx)
}

```

A module can be used through a relayer configuration file by registering a Config implementation for a target Chain or Prover in `RegisterInterfaces` method.

You can use it by specifying the package name of the proto definition corresponding to the Config in the "@type" field of the config file, as shown below. Then, the relayer creates an instance of the corresponding Chain or Prover using the Config at runtime.

```json
{
  "chain": {
    "@type": "/relayer.chains.tendermint.config.ChainConfig",
    "key": "testkey",
    "chain_id": "ibc0",
    "rpc_addr": "http://localhost:26657",
    "account_prefix": "cosmos",
    "gas_adjustment": 1.5,
    "gas_prices": "0.025stake"
  },
  "prover": {
    "@type": "/relayer.chains.tendermint.config.ProverConfig",
    "trusting_period": "336h"
  }
}
```
