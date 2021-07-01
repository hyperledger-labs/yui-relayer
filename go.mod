module github.com/datachainlab/relayer

go 1.15

require (
	github.com/avast/retry-go v2.6.0+incompatible
	github.com/cosmos/cosmos-sdk v0.40.0-rc3
	github.com/cosmos/go-bip39 v0.0.0-20180819234021-555e2067c45d
	github.com/gogo/protobuf v1.3.2
	github.com/hyperledger-labs/yui-corda-ibc/go v0.0.0-20210701073432-816924a5f8ce
	github.com/hyperledger-labs/yui-fabric-ibc v0.1.1-0.20210630081419-b3bd5a5cb1c7
	github.com/hyperledger/fabric v1.4.0-rc1.0.20200416031218-eff2f9306191
	github.com/hyperledger/fabric-protos-go v0.0.0-20200707132912-fee30f3ccd23
	github.com/hyperledger/fabric-sdk-go v1.0.0-beta2.0.20200715151216-87f5eb8a655f
	github.com/regen-network/cosmos-proto v0.3.1 // indirect
	github.com/spf13/cobra v1.1.1
	github.com/spf13/viper v1.7.1
	github.com/tendermint/tendermint v0.34.0-rc6
	github.com/tendermint/tm-db v0.6.2
	golang.org/x/sync v0.0.0-20190911185100-cd5d95a43a6e
	google.golang.org/grpc v1.35.0
	google.golang.org/protobuf v1.25.0
	gopkg.in/yaml.v2 v2.3.0
)

replace (
	github.com/go-kit/kit => github.com/go-kit/kit v0.8.0
	github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.2-alpha.regen.4
	github.com/hyperledger/fabric-sdk-go => github.com/datachainlab/fabric-sdk-go v0.0.0-20200730074927-282a61dcd92e
	github.com/keybase/go-keychain => github.com/99designs/go-keychain v0.0.0-20191008050251-8e49817e8af4
)
