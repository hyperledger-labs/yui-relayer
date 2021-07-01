module github.com/hyperledger-labs/yui-relayer/tests/chains/fabric/chaincode/fabibc

go 1.15

require (
	github.com/cosmos/cosmos-sdk v0.40.0-rc3
	github.com/datachainlab/fabric-ibc v0.0.0-20210118062001-0f577c6f6438
	github.com/hyperledger/fabric-chaincode-go v0.0.0-20200511190512-bcfeb58dd83a
	github.com/hyperledger/fabric-contract-api-go v1.1.0
	github.com/tendermint/tendermint v0.34.0-rc6
	github.com/tendermint/tm-db v0.6.2
)

replace (
	github.com/cosmos/cosmos-sdk => github.com/datachainlab/cosmos-sdk v0.34.4-0.20210118084708-b5c1fb2ebb3e
	github.com/go-kit/kit => github.com/go-kit/kit v0.8.0
	github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.2-alpha.regen.4
	github.com/keybase/go-keychain => github.com/99designs/go-keychain v0.0.0-20191008050251-8e49817e8af4
)
