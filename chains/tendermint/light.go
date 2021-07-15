package tendermint

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/avast/retry-go"
	tmclient "github.com/cosmos/ibc-go/modules/light-clients/07-tendermint/types"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/light"
	lightp "github.com/tendermint/tendermint/light/provider"
	lighthttp "github.com/tendermint/tendermint/light/provider/http"
	dbs "github.com/tendermint/tendermint/light/store/db"
	tmtypes "github.com/tendermint/tendermint/types"
	dbm "github.com/tendermint/tm-db"
)

// NOTE: currently we are discarding the very noisy light client logs
// it would be nice if we could add a setting the chain or otherwise
// that allowed users to enable light client logging. (maybe as a hidden prop
// on the Chain struct that users could pass in the config??)
var logger = light.Logger(log.NewTMLogger(log.NewSyncWriter(ioutil.Discard)))

// ErrLightNotInitialized returns the canonical error for a an uninitialized light client
var ErrLightNotInitialized = errors.New("light client is not initialized")

// LightClient initializes the light client for a given chain from the trusted store in the database
// this should be call for all other light client usage
func (pr *Prover) LightClient(db dbm.DB) (*light.Client, error) {
	prov := pr.LightHTTP()
	return light.NewClientFromTrustedStore(
		pr.chain.config.ChainId,
		pr.getTrustingPeriod(),
		prov,
		// TODO: provide actual witnesses!
		// NOTE: This requires adding them to the chain config
		[]lightp.Provider{prov},
		dbs.New(db, ""),
		logger,
	)
}

// LightHTTP returns the http client for light clients
func (pr *Prover) LightHTTP() lightp.Provider {
	cl, err := lighthttp.New(pr.chain.config.ChainId, pr.chain.config.RpcAddr)
	if err != nil {
		panic(err)
	}
	return cl
}

func (pr *Prover) NewLightDB() (db *dbm.GoLevelDB, df func(), err error) {
	c := pr.chain
	if err := retry.Do(func() error {
		db, err = dbm.NewGoLevelDB(c.config.ChainId, lightDir(c.HomePath))
		if err != nil {
			return fmt.Errorf("can't open light client database: %w", err)
		}
		return nil
	}, rtyAtt, rtyDel, rtyErr); err != nil {
		return nil, nil, err
	}

	df = func() {
		err := db.Close()
		if err != nil {
			panic(err)
		}
	}

	return
}

// DeleteLightDB removes the light client database on disk, forcing re-initialization
func (pr *Prover) DeleteLightDB() error {
	return os.RemoveAll(filepath.Join(lightDir(pr.chain.HomePath), fmt.Sprintf("%s.db", pr.chain.ChainID())))
}

// LightClientWithTrust takes a header from the chain and attempts to add that header to the light
// database.
func (pr *Prover) LightClientWithTrust(db dbm.DB, to light.TrustOptions) (*light.Client, error) {
	prov := pr.LightHTTP()
	return light.NewClient(
		context.Background(),
		pr.chain.config.ChainId,
		to,
		prov,
		// TODO: provide actual witnesses!
		// NOTE: This requires adding them to the chain config
		[]lightp.Provider{prov},
		dbs.New(db, ""),
		logger)
}

// LightClientWithoutTrust querys the latest header from the chain and initializes a new light client
// database using that header. This should only be called when first initializing the light client
func (pr *Prover) LightClientWithoutTrust(db dbm.DB) (*light.Client, error) {
	var (
		height int64
		err    error
	)
	prov := pr.LightHTTP()

	if err := retry.Do(func() error {
		height, err = pr.chain.GetLatestHeight()
		switch {
		case err != nil:
			return err
		case height == 0:
			return fmt.Errorf("shouldn't be here")
		default:
			return nil
		}
	}, rtyAtt, rtyDel, rtyErr); err != nil {
		return nil, err
	}

	lb, err := prov.LightBlock(context.Background(), height)
	if err != nil {
		return nil, err
	}
	return light.NewClient(
		context.Background(),
		pr.chain.config.ChainId,
		light.TrustOptions{
			Period: pr.getTrustingPeriod(),
			Height: height,
			Hash:   lb.SignedHeader.Hash(),
		},
		prov,
		// TODO: provide actual witnesses!
		// NOTE: This requires adding them to the chain config
		[]lightp.Provider{prov},
		dbs.New(db, ""),
		logger)
}

// GetLatestLightHeader returns the header to be used for client creation
func (pr *Prover) GetLatestLightHeader() (*tmclient.Header, error) {
	return pr.GetLightSignedHeaderAtHeight(0)
}

// GetLightSignedHeaderAtHeight returns a signed header at a particular height.
func (pr *Prover) GetLightSignedHeaderAtHeight(height int64) (*tmclient.Header, error) {
	// create database connection
	db, df, err := pr.NewLightDB()
	if err != nil {
		return nil, err
	}
	defer df()

	client, err := pr.LightClient(db)
	if err != nil {
		return nil, err
	}

	sh, err := client.TrustedLightBlock(height)
	if err != nil {
		return nil, err
	}

	protoVal, err := tmtypes.NewValidatorSet(sh.ValidatorSet.Validators).ToProto()
	if err != nil {
		return nil, err
	}

	return &tmclient.Header{SignedHeader: sh.SignedHeader.ToProto(), ValidatorSet: protoVal}, nil
}

func lightDir(home string) string {
	return path.Join(home, "light")
}
