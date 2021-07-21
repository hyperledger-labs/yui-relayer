package main

import (
	"log"

	corda "github.com/hyperledger-labs/yui-relayer/chains/corda/module"
	fabric "github.com/hyperledger-labs/yui-relayer/chains/fabric/module"
	tendermint "github.com/hyperledger-labs/yui-relayer/chains/tendermint/module"
	"github.com/hyperledger-labs/yui-relayer/cmd"
	mock "github.com/hyperledger-labs/yui-relayer/provers/mock/module"
)

func main() {
	if err := cmd.Execute(
		tendermint.Module{},
		fabric.Module{},
		corda.Module{},
		mock.Module{},
	); err != nil {
		log.Fatal(err)
	}
}
