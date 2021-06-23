module github.com/hyperledger-labs/yui-relayer/tests/tendermint

go 1.15

require (
	github.com/cosmos/cosmos-sdk v0.40.0-rc3
	github.com/datachainlab/cross v0.0.8-0.20210106062451-70e261f7aa8f
	github.com/datachainlab/fabric-ibc v0.0.0-20210118090849-c2eaee7a3314
	github.com/gogo/protobuf v1.3.1
	github.com/golang/protobuf v1.4.3
	github.com/gorilla/mux v1.8.0
	github.com/grpc-ecosystem/grpc-gateway v1.15.2
	github.com/rakyll/statik v0.1.7
	github.com/spf13/cast v1.3.1
	github.com/spf13/cobra v1.1.1
	github.com/spf13/viper v1.7.1
	github.com/stretchr/testify v1.6.1
	github.com/tendermint/tendermint v0.34.0-rc6
	github.com/tendermint/tm-db v0.6.2
	google.golang.org/genproto v0.0.0-20201111145450-ac7456db90a6
	google.golang.org/grpc v1.33.2
)

replace github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.2-alpha.regen.4
