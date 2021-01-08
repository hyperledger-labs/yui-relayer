module github.com/datachainlab/relayer

go 1.15

require (
	github.com/cosmos/cosmos-sdk v0.40.0-rc3
	github.com/cosmos/relayer v1.0.0-rc1
	github.com/datachainlab/fabric-ibc v0.0.0-20210108102243-566420dde41f
	github.com/gogo/protobuf v1.3.1
	github.com/hyperledger/fabric v1.4.0-rc1.0.20200416031218-eff2f9306191
	github.com/hyperledger/fabric-protos-go v0.0.0-20200707132912-fee30f3ccd23
	github.com/hyperledger/fabric-sdk-go v1.0.0-beta2.0.20200715151216-87f5eb8a655f
	github.com/spf13/cobra v1.1.1
	github.com/spf13/viper v1.7.1
	github.com/tendermint/tendermint v0.34.0-rc6
	golang.org/x/sync v0.0.0-20190911185100-cd5d95a43a6e
	gopkg.in/yaml.v2 v2.3.0
)

replace (
	github.com/go-kit/kit => github.com/go-kit/kit v0.8.0
	github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.2-alpha.regen.4
	github.com/hyperledger/fabric-sdk-go => github.com/datachainlab/fabric-sdk-go v0.0.0-20200730074927-282a61dcd92e
	github.com/keybase/go-keychain => github.com/99designs/go-keychain v0.0.0-20191008050251-8e49817e8af4
)
