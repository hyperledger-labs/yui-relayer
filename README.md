# Relayer

![Test](https://github.com/hyperledger-labs/yui-relayer/workflows/Test/badge.svg)
[![GoDoc](https://godoc.org/github.com/hyperledger-labs/yui-relayer?status.svg)](https://pkg.go.dev/github.com/hyperledger-labs/yui-relayer?tab=doc)

A relayer implementation supports packet relays between various blockchains.

Currently supports:
- Cosmos/Tendermint([ibc-go](https://github.com/cosmos/ibc-go))
  - This implementation is a fork of [cosmos/relayer](https://github.com/cosmos/relayer)
- Hyperledger Fabric([fabric-ibc](https://github.com/hyperledger-labs/yui-fabric-ibc))
- Corda([corda-ibc](https://github.com/hyperledger-labs/yui-corda-ibc))
