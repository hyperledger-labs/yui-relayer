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

| Relayer                                                                     | ibc-go |
|-----------------------------------------------------------------------------|--------|
| v0.5(current branch)                                                        | v8     |
| [v0.4](https://github.com/hyperledger-labs/yui-relayer/releases/tag/v0.4.0) | v7     |
| [v0.3](https://github.com/hyperledger-labs/yui-relayer/releases/tag/v0.3.0) | v4     |
| [v0.2](https://github.com/hyperledger-labs/yui-relayer/releases/tag/v0.2.0) | v1     |

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

## OpenTelemetry integration

OpenTelemetry integration can be enabled by specifying the `--enable-telemetry` flag or by setting `YLRY_ENABLE_TELEMETRY` environment variable to true.
To see an example setup, refer to [examples/opentelemetry-integration](examples/opentelemetry-integration).

### Configurations

You can configure its behavior using environment variables supported by Go, as listed in the [Compliance of Implementations with Specification](https://github.com/open-telemetry/opentelemetry-specification/blob/main/spec-compliance-matrix.md#environment-variables).

In addition to the environment variables supported by Go, yui-relayer supports the following variables, which are not available in the Go SDK:

* OTEL_PROPAGATORS
* OTEL_TRACES_EXPORTER
    - Note that `"zipkin"` is not supported
* OTEL_METRICS_EXPORTER
* OTEL_LOGS_EXPORTER
* OTEL_EXPORTER_PROMETHEUS_HOST
* OTEL_EXPORTER_PROMETHEUS_PORT
* OTEL_EXPORTER_CONSOLE_TRACES_WRITER
* OTEL_EXPORTER_CONSOLE_LOGS_WRITER
* OTEL_EXPORTER_CONSOLE_METRICS_WRITER

The `OTEL_EXPORTER_CONSOLE_*_WRITER` variables are specific to yui-relayer and allow you to change the output destination of the standard output exporters. To redirect output to standard error, set the value to `stderr`.

For more information about OpenTelemetry environment variables, refer to the [OpenTelemetry Environment Variable Specification](https://opentelemetry.io/docs/specs/otel/configuration/sdk-environment-variables).


When OpenTelemetry integration is enabled, the OTLP log exporter is enabled by default and you may want to disable ordinal logs.
In this case, you can disable them by setting `.global.logger.output` to `"null"` in the yui-relayer configuration file.

### Add spans and span attributes in external modules

#### Using tracing bridges

The Relayer provides OpenTelemetry tracing bridges: `otelcore.Chain` and `otelcore.Prover`.
These bridges add tracing to the primary methods defined in the Chain and Prover interfaces.
You can use the tracing bridges by returning them in `ChainConfig.Build` and `ProverConfig.Build`:

```go
var tracer = otel.Tracer("example.com/my-module")

func (c ChainConfig) Build() (core.Chain, error) {
	return otelcore.NewChain(&Chain{}, tracer), nil
}

func (c ProverConfig) Build(chain core.Chain) (core.Prover, error) {
	return otelcore.NewProver(&Prover{}, chain.ChainID(), tracer), nil
}
```

If you need to acces the original Chain and Prover implementations, you can unwrap them as follows:

```go
chain, err := otelcore.UnwrapChain(provableChain.Chain)
originalChain := chain.(module.Chain)
prover, err := otelcore.UnwrapProver(provableChain.Prover)
originalProver := prover.(module.Prover)
```

Alternatively, you can use `core.AsChain` and `core.AsProver`, similar to how `errors.As` works:

```go
var chain module.Chain
ok := core.AsChain(provableChain, &chain)
var prover module.Prover
ok := core.AsProver(provableChain, &prover)
```

Note that, if you call methods defined in your Chain module and Prover module directly, tracing data will not be recorded.

#### Manual tracing

In addition to using the tracing bridges, you can manually create spans when needed:

```go
var tracer = otel.Tracer("example.com/my-module")

func someFunction(ctx context.Context) {
	ctx, span := tracer.Start(ctx, "someFunction")
	defer span.End()

	// -- snip --
}
```

If a function or method receives a `core.QueryContext`, you can use `core.StartTraceWithQueryContext` to create a span:

```go
func (c *Chain) QuerySomething(ctx core.QueryContext) (any, error) {
	ctx, span := core.StartTraceWithQueryContext(tracer, ctx, "Chain.QuerySomething", core.WithChainAttributes(c.ChainID()))
	defer span.End()

	// -- snip --
```

You can also add span attributes as follows:

```go
func (c *Chain) GetMsgResult(ctx context.Context, id core.MsgID) (core.MsgResult, error) {
	msgID, ok := id.(*MsgID)
	if !ok {
		return nil, fmt.Errorf("unexpected message id type: %T", id)
	}

	span := trace.SpanFromContext(ctx)
	span.SetAttributes(core.AttributeKeyTxHash.String(msgID.TxHash))

	// -- snip --
}
```
