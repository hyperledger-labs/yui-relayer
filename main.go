package main

import (
	corda "github.com/hyperledger-labs/yui-relayer/chains/corda/module"
	fabric "github.com/hyperledger-labs/yui-relayer/chains/fabric/module"
	tendermint "github.com/hyperledger-labs/yui-relayer/chains/tendermint/module"
	"github.com/hyperledger-labs/yui-relayer/cmd"
)

func main() {
	cmd.Execute(
		tendermint.Module{},
		fabric.Module{},
		corda.Module{},
	)
}
