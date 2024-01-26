package main

import (
	"os"

	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"

	"github.com/hyperledger-labs/yui-relayer/tests/chains/tendermint/simapp"
	"github.com/hyperledger-labs/yui-relayer/tests/chains/tendermint/simapp/simd/cmd"
)

func main() {
	rootCmd := cmd.NewRootCmd()

	if err := svrcmd.Execute(rootCmd, "simd", simapp.DefaultNodeHome); err != nil {
		os.Exit(1)
	}
}
