package main

import (
	"fmt"
	"io"
	"os"

	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	ibc "github.com/cosmos/ibc-go/modules/core"
	ibckeeper "github.com/cosmos/ibc-go/modules/core/keeper"
	ibctypes "github.com/cosmos/ibc-go/modules/core/types"
	corda "github.com/hyperledger-labs/yui-corda-ibc/go/x/ibc/light-clients/xx-corda"
	cordatypes "github.com/hyperledger-labs/yui-corda-ibc/go/x/ibc/light-clients/xx-corda/types"
	"github.com/hyperledger-labs/yui-fabric-ibc/app"
	"github.com/hyperledger-labs/yui-fabric-ibc/chaincode"
	"github.com/hyperledger-labs/yui-fabric-ibc/commitment"
	"github.com/hyperledger-labs/yui-fabric-ibc/example"
	fabricauthante "github.com/hyperledger-labs/yui-fabric-ibc/x/auth/ante"
	fabrictypes "github.com/hyperledger-labs/yui-fabric-ibc/x/ibc/light-clients/xx-fabric/types"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
	tmjson "github.com/tendermint/tendermint/libs/json"
	tmlog "github.com/tendermint/tendermint/libs/log"
	tmdb "github.com/tendermint/tm-db"
)

func init() {
	ms := []module.AppModuleBasic{corda.AppModuleBasic{}}
	for _, m := range example.ModuleBasics {
		ms = append(ms, m)
	}
	example.ModuleBasics = module.NewBasicManager(ms...)
}

func main() {
	cc := chaincode.NewIBCChaincode(
		"fabricibc",
		tmlog.NewTMLogger(os.Stdout),
		commitment.NewDefaultSequenceManager(),
		newApp,
		anteHandler,
		chaincode.DefaultDBProvider,
		chaincode.DefaultMultiEventHandler(),
	)
	chaincode, err := contractapi.NewChaincode(cc)
	if err != nil {
		panic(err)
	}

	server := &shim.ChaincodeServer{
		CCID:    os.Getenv("CHAINCODE_CCID"),
		Address: os.Getenv("CHAINCODE_ADDRESS"),
		CC:      chaincode,
		TLSProps: shim.TLSProperties{
			Disabled: true,
		},
	}
	if err = server.Start(); err != nil {
		fmt.Printf("Error starting IBC chaincode: %s", err)
	}
}

func newApp(appName string, logger tmlog.Logger, db tmdb.DB, traceStore io.Writer, seqMgr commitment.SequenceManager, blockProvider app.BlockProvider, anteHandlerProvider app.AnteHandlerProvider) (app.Application, error) {
	app, err := example.NewIBCApp(
		appName,
		logger,
		db,
		traceStore,
		example.MakeEncodingConfig(),
		seqMgr,
		blockProvider,
		anteHandlerProvider,
	)
	if err != nil {
		return nil, err
	}
	wrapped := &IBCApp{IBCApp: app}
	app.SetInitChainer(wrapped.InitChainer)
	return wrapped, nil
}

func anteHandler(
	ibcKeeper ibckeeper.Keeper,
	sigGasConsumer ante.SignatureVerificationGasConsumer,
) sdk.AnteHandler {
	return sdk.ChainAnteDecorators(
		ante.NewValidateBasicDecorator(),
		fabricauthante.NewFabricIDVerificationDecorator(),
	)
}

type IBCApp struct {
	*example.IBCApp
}

func (app *IBCApp) InitChainer(ctx sdk.Context, appStateBytes []byte) error {
	var genesisState simapp.GenesisState
	if err := tmjson.Unmarshal(appStateBytes, &genesisState); err != nil {
		return err
	}
	ibcGenesisState := ibctypes.DefaultGenesisState()
	ibcGenesisState.ClientGenesis.Params.AllowedClients = append(ibcGenesisState.ClientGenesis.Params.AllowedClients, fabrictypes.Fabric, cordatypes.CordaClientType)
	genesisState[ibc.AppModule{}.Name()] = app.AppCodec().MustMarshalJSON(ibcGenesisState)
	bz, err := tmjson.Marshal(genesisState)
	if err != nil {
		return err
	}
	return app.IBCApp.InitChainer(ctx, bz)
}
