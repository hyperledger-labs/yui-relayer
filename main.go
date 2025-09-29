package main

import (
	"log"

	debug_chain "github.com/hyperledger-labs/yui-relayer/chains/debug/module"
	tendermint "github.com/hyperledger-labs/yui-relayer/chains/tendermint/module"
	"github.com/hyperledger-labs/yui-relayer/cmd"
	debug_prover "github.com/hyperledger-labs/yui-relayer/provers/debug/module"
	mock "github.com/hyperledger-labs/yui-relayer/provers/mock/module"
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
