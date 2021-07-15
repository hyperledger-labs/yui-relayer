package main

import (
	"log"

	corda "github.com/hyperledger-labs/yui-relayer/chains/corda/module"
	fabric "github.com/hyperledger-labs/yui-relayer/chains/fabric/module"
	tendermint "github.com/hyperledger-labs/yui-relayer/chains/tendermint/module"
	"github.com/hyperledger-labs/yui-relayer/cmd"
)

func main() {
	if err := cmd.Execute(
		tendermint.Module{},
		fabric.Module{},
		corda.Module{},
	); err != nil {
		log.Fatal(err)
	}
}
