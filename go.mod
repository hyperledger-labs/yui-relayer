module github.com/datachainlab/relayer

go 1.15

require (
	github.com/cosmos/cosmos-sdk v0.40.0-rc3
	github.com/cosmos/relayer v1.0.0-rc1
	github.com/gogo/protobuf v1.3.1
	github.com/spf13/cobra v1.1.1
	github.com/spf13/viper v1.7.1
	github.com/tendermint/tendermint v0.34.0-rc6
	golang.org/x/sync v0.0.0-20190911185100-cd5d95a43a6e
	gopkg.in/yaml.v2 v2.3.0
)

replace github.com/keybase/go-keychain => github.com/99designs/go-keychain v0.0.0-20191008050251-8e49817e8af4

replace github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.2-alpha.regen.4
