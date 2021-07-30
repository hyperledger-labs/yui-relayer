package tendermint

import (
	"context"
	"fmt"
	"os"
	"path"
	"sync"
	"time"

	"github.com/avast/retry-go"
	sdkCtx "github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/codec"
	keys "github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authTypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/go-bip39"
	"github.com/tendermint/tendermint/libs/log"
	rpcclient "github.com/tendermint/tendermint/rpc/client"
	rpchttp "github.com/tendermint/tendermint/rpc/client/http"
	libclient "github.com/tendermint/tendermint/rpc/jsonrpc/client"

	"github.com/hyperledger-labs/yui-relayer/core"
)

var (
	rtyAttNum = uint(5)
	rtyAtt    = retry.Attempts(rtyAttNum)
	rtyDel    = retry.Delay(time.Millisecond * 400)
	rtyErr    = retry.LastErrorOnly(true)
)

// Chain represents the necessary data for connecting to and indentifying a chain and its counterparites
type Chain struct {
	config ChainConfig

	// TODO: make these private
	HomePath string           `yaml:"-" json:"-"`
	PathEnd  *core.PathEnd    `yaml:"-" json:"-"`
	Keybase  keys.Keyring     `yaml:"-" json:"-"`
	Client   rpcclient.Client `yaml:"-" json:"-"`

	codec codec.ProtoCodecMarshaler `yaml:"-" json:"-"`

	address sdk.AccAddress
	logger  log.Logger
	timeout time.Duration
	debug   bool

	// stores facuet addresses that have been used reciently
	faucetAddrs map[string]time.Time
}

var _ core.ChainI = (*Chain)(nil)

func (c *Chain) ChainID() string {
	return c.config.ChainId
}

func (c *Chain) Config() ChainConfig {
	return c.config
}

func (c *Chain) ClientID() string {
	return c.PathEnd.ClientID
}

func (c *Chain) Codec() codec.ProtoCodecMarshaler {
	return c.codec
}

// GetAddress returns the sdk.AccAddress associated with the configred key
func (c *Chain) GetAddress() (sdk.AccAddress, error) {
	defer c.UseSDKContext()()
	if c.address != nil {
		return c.address, nil
	}

	// Signing key for c chain
	srcAddr, err := c.Keybase.Key(c.config.Key)
	if err != nil {
		return nil, err
	}

	return srcAddr.GetAddress(), nil
}

// SetPath sets the path and validates the identifiers
func (c *Chain) SetPath(p *core.PathEnd) error {
	err := p.Validate()
	if err != nil {
		return c.ErrCantSetPath(err)
	}
	c.PathEnd = p
	return nil
}

// ErrCantSetPath returns an error if the path doesn't set properly
func (c *Chain) ErrCantSetPath(err error) error {
	return fmt.Errorf("path on chain %s failed to set: %w", c.ChainID(), err)
}

func (c *Chain) Path() *core.PathEnd {
	return c.PathEnd
}

func (c *Chain) Init(homePath string, timeout time.Duration, codec codec.ProtoCodecMarshaler, debug bool) error {
	keybase, err := keys.New(c.config.ChainId, "test", keysDir(homePath, c.config.ChainId), nil)
	if err != nil {
		return err
	}

	client, err := newRPCClient(c.config.RpcAddr, timeout)
	if err != nil {
		return err
	}

	_, err = sdk.ParseDecCoins(c.config.GasPrices)
	if err != nil {
		return fmt.Errorf("failed to parse gas prices (%s) for chain %s", c.config.GasPrices, c.ChainID())
	}

	c.Keybase = keybase
	c.Client = client
	c.HomePath = homePath
	c.codec = codec
	c.logger = defaultChainLogger()
	c.timeout = timeout
	c.debug = debug
	c.faucetAddrs = make(map[string]time.Time)
	return nil
}

// QueryLatestHeight queries the chain for the latest height and returns it
func (c *Chain) GetLatestHeight() (int64, error) {
	res, err := c.Client.Status(context.Background())
	if err != nil {
		return -1, err
	} else if res.SyncInfo.CatchingUp {
		return -1, fmt.Errorf("node at %s running chain %s not caught up", c.config.RpcAddr, c.ChainID())
	}

	return res.SyncInfo.LatestBlockHeight, nil
}

func (c *Chain) sendMsgs(msgs []sdk.Msg) (*sdk.TxResponse, error) {
	res, _, err := c.rawSendMsgs(msgs)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (c *Chain) rawSendMsgs(msgs []sdk.Msg) (*sdk.TxResponse, bool, error) {
	// Instantiate the client context
	ctx := c.CLIContext(0)

	// Query account details
	txf, err := prepareFactory(ctx, c.TxFactory(0))
	if err != nil {
		return nil, false, err
	}

	// TODO: Make this work with new CalculateGas method
	// https://github.com/cosmos/cosmos-sdk/blob/5725659684fc93790a63981c653feee33ecf3225/client/tx/tx.go#L297
	// If users pass gas adjustment, then calculate gas
	_, adjusted, err := CalculateGas(ctx.QueryWithData, txf, msgs...)
	if err != nil {
		return nil, false, err
	}

	// Set the gas amount on the transaction factory
	txf = txf.WithGas(adjusted)

	// Build the transaction builder
	txb, err := tx.BuildUnsignedTx(txf, msgs...)
	if err != nil {
		return nil, false, err
	}

	// Attach the signature to the transaction
	err = tx.Sign(txf, c.config.Key, txb, false)
	if err != nil {
		return nil, false, err
	}

	// Generate the transaction bytes
	txBytes, err := ctx.TxConfig.TxEncoder()(txb.GetTx())
	if err != nil {
		return nil, false, err
	}

	// Broadcast those bytes
	res, err := ctx.BroadcastTx(txBytes)
	if err != nil {
		return nil, false, err
	}

	// transaction was executed, log the success or failure using the tx response code
	// NOTE: error is nil, logic should use the returned error to determine if the
	// transaction was successfully executed.
	if res.Code != 0 {
		c.LogFailedTx(res, err, msgs)
		return res, false, nil
	}

	c.LogSuccessTx(res, msgs)
	return res, true, nil
}

func prepareFactory(clientCtx sdkCtx.Context, txf tx.Factory) (tx.Factory, error) {
	from := clientCtx.GetFromAddress()

	if err := txf.AccountRetriever().EnsureExists(clientCtx, from); err != nil {
		return txf, err
	}

	initNum, initSeq := txf.AccountNumber(), txf.Sequence()
	if initNum == 0 || initSeq == 0 {
		num, seq, err := txf.AccountRetriever().GetAccountNumberSequence(clientCtx, from)
		if err != nil {
			return txf, err
		}

		if initNum == 0 {
			txf = txf.WithAccountNumber(num)
		}

		if initSeq == 0 {
			txf = txf.WithSequence(seq)
		}
	}

	return txf, nil
}

// protoTxProvider is a type which can provide a proto transaction. It is a
// workaround to get access to the wrapper TxBuilder's method GetProtoTx().
type protoTxProvider interface {
	GetProtoTx() *txtypes.Tx
}

// BuildSimTx creates an unsigned tx with an empty single signature and returns
// the encoded transaction or an error if the unsigned transaction cannot be
// built.
func BuildSimTx(txf tx.Factory, msgs ...sdk.Msg) ([]byte, error) {
	txb, err := tx.BuildUnsignedTx(txf, msgs...)
	if err != nil {
		return nil, err
	}

	// Create an empty signature literal as the ante handler will populate with a
	// sentinel pubkey.
	sig := signing.SignatureV2{
		PubKey: &secp256k1.PubKey{},
		Data: &signing.SingleSignatureData{
			SignMode: txf.SignMode(),
		},
		Sequence: txf.Sequence(),
	}
	if err := txb.SetSignatures(sig); err != nil {
		return nil, err
	}

	protoProvider, ok := txb.(protoTxProvider)
	if !ok {
		return nil, fmt.Errorf("cannot simulate amino tx")
	}
	simReq := txtypes.SimulateRequest{Tx: protoProvider.GetProtoTx()}

	return simReq.Marshal()
}

// CalculateGas simulates the execution of a transaction and returns the
// simulation response obtained by the query and the adjusted gas amount.
func CalculateGas(
	queryFunc func(string, []byte) ([]byte, int64, error), txf tx.Factory, msgs ...sdk.Msg,
) (txtypes.SimulateResponse, uint64, error) {
	txBytes, err := BuildSimTx(txf, msgs...)
	if err != nil {
		return txtypes.SimulateResponse{}, 0, err
	}

	bz, _, err := queryFunc("/cosmos.tx.v1beta1.Service/Simulate", txBytes)
	if err != nil {
		return txtypes.SimulateResponse{}, 0, err
	}

	var simRes txtypes.SimulateResponse

	if err := simRes.Unmarshal(bz); err != nil {
		return txtypes.SimulateResponse{}, 0, err
	}

	return simRes, uint64(txf.GasAdjustment() * float64(simRes.GasInfo.GasUsed)), nil
}

func (c *Chain) SendMsgs(msgs []sdk.Msg) ([]byte, error) {
	// Broadcast those bytes
	res, err := c.sendMsgs(msgs)
	if err != nil {
		return nil, err
	}
	return []byte(res.Logs.String()), nil
}

func (c *Chain) Send(msgs []sdk.Msg) bool {
	res, err := c.sendMsgs(msgs)
	if err != nil || res.Code != 0 {
		c.LogFailedTx(res, err, msgs)
		return false
	}
	// NOTE: Add more data to this such as identifiers
	c.LogSuccessTx(res, msgs)

	return true
}

func (c *Chain) StartEventListener(dst core.ChainI, strategy core.StrategyI) {
	panic("not implemented error")
}

// ------------------------------- //

func (c *Chain) Key() string {
	return c.config.Key
}

// KeyExists returns true if there is a specified key in chain's keybase
func (c *Chain) KeyExists(name string) bool {
	k, err := c.Keybase.Key(name)
	if err != nil {
		return false
	}

	return k.GetName() == name
}

// MustGetAddress used for brevity
func (c *Chain) MustGetAddress() sdk.AccAddress {
	srcAddr, err := c.GetAddress()
	if err != nil {
		panic(err)
	}
	return srcAddr
}

var sdkContextMutex sync.Mutex

// UseSDKContext uses a custom Bech32 account prefix and returns a restore func
// CONTRACT: When using this function, caller must ensure that lock contention
// doesn't cause program to hang. This function is only for use in codec calls
func (c *Chain) UseSDKContext() func() {
	// Ensure we're the only one using the global context,
	// lock context to begin function
	sdkContextMutex.Lock()

	// Mutate the sdkConf
	sdkConf := sdk.GetConfig()
	sdkConf.SetBech32PrefixForAccount(c.config.AccountPrefix, c.config.AccountPrefix+"pub")
	sdkConf.SetBech32PrefixForValidator(c.config.AccountPrefix+"valoper", c.config.AccountPrefix+"valoperpub")
	sdkConf.SetBech32PrefixForConsensusNode(c.config.AccountPrefix+"valcons", c.config.AccountPrefix+"valconspub")

	// Return the unlock function, caller must lock and ensure that lock is released
	// before any other function needs to use c.UseSDKContext
	return sdkContextMutex.Unlock
}

// CLIContext returns an instance of client.Context derived from Chain
func (c *Chain) CLIContext(height int64) sdkCtx.Context {
	return sdkCtx.Context{}.
		WithChainID(c.config.ChainId).
		WithJSONCodec(newContextualStdCodec(c.codec, c.UseSDKContext)).
		WithInterfaceRegistry(c.codec.InterfaceRegistry()).
		WithTxConfig(authtx.NewTxConfig(c.codec, authtx.DefaultSignModes)).
		WithInput(os.Stdin).
		WithNodeURI(c.config.RpcAddr).
		WithClient(c.Client).
		WithAccountRetriever(authTypes.AccountRetriever{}).
		WithBroadcastMode(flags.BroadcastBlock).
		WithKeyring(c.Keybase).
		WithOutputFormat("json").
		WithFrom(c.config.Key).
		WithFromName(c.config.Key).
		WithFromAddress(c.MustGetAddress()).
		WithSkipConfirmation(true).
		WithNodeURI(c.config.RpcAddr).
		WithHeight(height)
}

// TxFactory returns an instance of tx.Factory derived from
func (c *Chain) TxFactory(height int64) tx.Factory {
	ctx := c.CLIContext(height)
	return tx.Factory{}.
		WithAccountRetriever(ctx.AccountRetriever).
		WithChainID(c.config.ChainId).
		WithTxConfig(ctx.TxConfig).
		WithGasAdjustment(c.config.GasAdjustment).
		WithGasPrices(c.config.GasPrices).
		WithKeybase(c.Keybase).
		WithSignMode(signing.SignMode_SIGN_MODE_DIRECT)
}

// KeysDir returns the path to the keys for this chain
func keysDir(home, chainID string) string {
	return path.Join(home, "keys", chainID)
}

func newRPCClient(addr string, timeout time.Duration) (*rpchttp.HTTP, error) {
	httpClient, err := libclient.DefaultHTTPClient(addr)
	if err != nil {
		return nil, err
	}

	httpClient.Timeout = timeout
	rpcClient, err := rpchttp.NewWithClient(addr, "/websocket", httpClient)
	if err != nil {
		return nil, err
	}

	return rpcClient, nil
}

func defaultChainLogger() log.Logger {
	return log.NewTMLogger(log.NewSyncWriter(os.Stdout))
}

// CreateMnemonic creates a new mnemonic
func CreateMnemonic() (string, error) {
	entropySeed, err := bip39.NewEntropy(256)
	if err != nil {
		return "", err
	}
	mnemonic, err := bip39.NewMnemonic(entropySeed)
	if err != nil {
		return "", err
	}
	return mnemonic, nil
}
