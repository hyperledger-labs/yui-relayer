# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

yui-relayer is an IBC relayer implementation (binary name: `yrly`) that supports heterogeneous blockchains — Cosmos/Tendermint (ibc-go), EVM chains (ibc-solidity), Hyperledger Fabric, Corda, etc. The Tendermint support is a fork of cosmos/relayer. Versioning: "v0.Y.Z" where Y tracks the ibc-go major version (current branch v0.5 ↔ ibc-go v8).

## Commands

```sh
make build        # builds ./build/yrly
make test         # runs `go generate ./...` (mockgen) then `go test -v ./...`
make pre-commit   # go mod tidy + go fmt ./... (CI fails if this leaves a diff)
make proto-gen    # regenerate Go code from proto/ (requires Docker; uses cosmos proto-builder image)
```

Run a single test:

```sh
go generate ./...                       # needed once: core tests depend on generated core/mock_chain_test.go
go test -v ./core/ -run TestName
```

### E2E tests (tests/)

CI runs E2E cases under `tests/cases/` (e.g. `tm2tm`, `tmmock2tmmock`) against Docker-based Tendermint chains. Requires Docker, `expect`, `jq`, and the relayer binary at `./build/yrly`.

```sh
cd tests/chains/tendermint && make docker-images   # build chain images (once)
cd tests/cases/tm2tm
make network       # docker compose up chains
make test          # runs scripts: fixture, init-rly, handshake, test-tx, test-service, ...
make network-down
```

## Architecture

### Module system (pluggable chains/provers)

The relayer supports new chains without forking: it is used as a library, and chain/prover support is provided via modules implementing `config.ModuleI` (`config/module.go`): `Name()`, `RegisterInterfaces(registry)` (registers `ChainConfig`/`ProverConfig` protobuf implementations), and `GetCmd(ctx)` (chain-specific cobra subcommands). `main.go` passes modules to `cmd.Execute(...)`; built-in modules are `chains/tendermint`, `chains/debug`, `provers/mock`, `provers/debug`.

Chains and provers are instantiated at runtime from the JSON config file: the `"@type"` field (protobuf Any type URL, e.g. `/relayer.chains.tendermint.config.ChainConfig`) selects a registered config type, and its `Build()` method constructs the `Chain`/`Prover`. Proto definitions for configs live in `proto/relayer/...`; generated code goes next to the consuming Go packages.

### Core abstractions (core/)

- **`Chain`** (`core/chain.go`): sends transactions (`SendMsgs`/`GetMsgResult`) and queries chain state (IBC state, packets, balances). Deliberately unaware of finality.
- **`Prover`** (`core/provers.go`): `LightClient` (create/update client states, `GetLatestFinalizedHeader`, `SetupHeadersForUpdate` returning a header stream channel) + `StateProver` (`ProveState`). Finality decisions live here, not in `Chain`.
- **`ProvableChain`** (`core/provable-chain.go`): struct embedding one `Chain` + one `Prover`; this is what the rest of core operates on.
- **`PathEnd`/`Path`** (`core/pathEnd.go`, `core/path.go`): the client/connection/channel identifiers on each side of a relay path.
- **`StrategyI`** (`core/strategies.go`): computes unrelayed packets/acks/timeouts and builds relay messages. Only implementation: `naive` (`core/naive-strategy.go`).
- **`RelayService`** (`core/service.go`): the `service start` polling loop — syncs headers (`SyncHeaders` in `core/headers.go`), asks the strategy for unrelayed packets/acks, optionally batches ("relay optimization" via interval/count thresholds), and sends msgs.
- Handshake logic (`client.go`, `connection.go`, `channel.go`, `channel-upgrade.go`) implements `tx clients/connection/channel/channel-upgrade` as retriable state machines that inspect both ends and send the next handshake msg.

### CLI (cmd/)

Cobra commands: `config`, `chains`, `paths`, `tx` (handshake + relay), `query`, `service`, `transfer`, `modules`. Global flags bind to viper with env prefix `YRLY_` (e.g. `--enable-telemetry` ↔ `YRLY_ENABLE_TELEMETRY`). Default home is `$HOME/.yui-relayer` with config at `config/config.json`.

### Supporting packages

- **`otelcore/`**: OpenTelemetry tracing bridges (`otelcore.NewChain`/`NewProver`) that wrap Chain/Prover; modules opt in by returning them from `Build()`. `coreutil.UnwrapChain`/`UnwrapProver` recover the concrete implementation from a wrapped `ProvableChain`.
- **`signer/`**: generic `Signer`/`SignerConfig` interfaces so chain modules can plug in alternative signing backends.
- **`log/`**: slog-based `RelayLogger` used throughout core.
- **`internal/telemetry`**: OTel setup (traces/metrics/logs exporters configured via `OTEL_*` env vars).
