module github.com/hyperledger-labs/yui-relayer/tests/tendermint

go 1.15

require (
	github.com/cosmos/cosmos-sdk v0.43.0-beta1
	github.com/cosmos/ibc-go v1.0.0-beta1
	github.com/datachainlab/ibc-mock-client v0.0.0-20210801010718-05f8b1087574 // indirect
	github.com/gorilla/mux v1.8.0
	github.com/hyperledger-labs/yui-fabric-ibc v0.2.0
	github.com/rakyll/statik v0.1.7
	github.com/spf13/cast v1.3.1
	github.com/spf13/cobra v1.1.3
	github.com/spf13/viper v1.7.1
	github.com/stretchr/testify v1.7.0
	github.com/tendermint/tendermint v0.34.10
	github.com/tendermint/tm-db v0.6.4
)

replace github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.2-alpha.regen.4
