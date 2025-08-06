package main

import (
	"log"

	tendermint "github.com/hyperledger-labs/yui-relayer/chains/tendermint/module"
	debug_chain "github.com/hyperledger-labs/yui-relayer/chains/debug/module"
	"github.com/hyperledger-labs/yui-relayer/cmd"
	mock "github.com/hyperledger-labs/yui-relayer/provers/mock/module"
	debug_prover "github.com/hyperledger-labs/yui-relayer/provers/debug/module"
)

func main() {
	if err := cmd.Execute(
		tendermint.Module{},
		mock.Module{},
		debug_chain.Module{},
		debug_prover.Module{},
	); err != nil {
		log.Fatal(err)
	}
}
