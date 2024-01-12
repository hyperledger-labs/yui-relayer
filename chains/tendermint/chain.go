package tendermint

import (
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"cosmossdk.io/errors"
	"github.com/avast/retry-go"

	rpcclient "github.com/cometbft/cometbft/rpc/client"
	rpchttp "github.com/cometbft/cometbft/rpc/client/http"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	libclient "github.com/cometbft/cometbft/rpc/jsonrpc/client"
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
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	ibcexported "github.com/cosmos/ibc-go/v7/modules/core/exported"

	"github.com/hyperledger-labs/yui-relayer/core"
	"github.com/hyperledger-labs/yui-relayer/log"
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

	codec            codec.ProtoCodecMarshaler `yaml:"-" json:"-"`
	msgEventListener core.MsgEventListener

	timeout time.Duration
	debug   bool

	// stores facuet addresses that have been used reciently
	faucetAddrs map[string]time.Time
}

var _ core.Chain = (*Chain)(nil)

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

	// Signing key for c chain
	srcAddr, err := c.Keybase.Key(c.config.Key)
	if err != nil {
		return nil, err
	}

	return srcAddr.GetAddress()
}

// SetRelayInfo sets source's path and counterparty's info to the chain
func (c *Chain) SetRelayInfo(p *core.PathEnd, _ *core.ProvableChain, _ *core.PathEnd) error {
	if err := p.Validate(); err != nil {
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
	keybase, err := keys.New(c.config.ChainId, "test", keysDir(homePath, c.config.ChainId), nil, codec)
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
	c.timeout = timeout
	c.debug = debug
	c.faucetAddrs = make(map[string]time.Time)
	return nil
}

func (c *Chain) SetupForRelay(ctx context.Context) error {
	return nil
}

// LatestHeight queries the chain for the latest height and returns it
func (c *Chain) LatestHeight() (ibcexported.Height, error) {
	res, err := c.Client.Status(context.Background())
	if err != nil {
		return nil, err
	} else if res.SyncInfo.CatchingUp {
		return nil, fmt.Errorf("node at %s running chain %s not caught up", c.config.RpcAddr, c.ChainID())
	}
	version := clienttypes.ParseChainID(c.ChainID())
	return clienttypes.NewHeight(version, uint64(res.SyncInfo.LatestBlockHeight)), nil
}

func (c *Chain) Timestamp(height ibcexported.Height) (time.Time, error) {
	ht := int64(height.GetRevisionHeight())
	if header, err := c.Client.Header(context.TODO(), &ht); err != nil {
		return time.Time{}, err
	} else {
		return header.Header.Time, nil
	}
}

func (c *Chain) AverageBlockTime() time.Duration {
	return time.Duration(c.config.AverageBlockTimeMsec) * time.Millisecond
}

// RegisterMsgEventListener registers a given EventListener to the chain
func (c *Chain) RegisterMsgEventListener(listener core.MsgEventListener) {
	c.msgEventListener = listener
}

func (c *Chain) sendMsgs(msgs []sdk.Msg) (*sdk.TxResponse, error) {
	logger := GetChainLogger()
	// broadcast tx
	res, _, err := c.rawSendMsgs(msgs)
	if err != nil {
		return nil, err
	} else if res.Code != 0 {
		// CheckTx failed
		return nil, fmt.Errorf("CheckTx failed: %v", errors.ABCIError(res.Codespace, res.Code, res.RawLog))
	}

	// wait for tx being committed
	if resTx, err := c.waitForCommit(res.TxHash); err != nil {
		return nil, err
	} else if resTx.TxResult.IsErr() {
		// DeliverTx failed
		return nil, fmt.Errorf("DeliverTx failed: %v", errors.ABCIError(res.Codespace, res.Code, res.RawLog))
	}

	// call msgEventListener if needed
	if c.msgEventListener != nil {
		if err := c.msgEventListener.OnSentMsg(msgs); err != nil {
			logger.Error("failed to OnSendMsg call", err)
			return res, nil
		}
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
	txb, err := txf.BuildUnsignedTx(msgs...)
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

func (c *Chain) waitForCommit(txHash string) (*coretypes.ResultTx, error) {
	var resTx *coretypes.ResultTx

	retryInterval := c.AverageBlockTime()
	maxRetry := uint(c.config.MaxRetryForCommit)

	if err := retry.Do(func() error {
		var err error
		var recoverable bool
		resTx, recoverable, err = c.rawQueryTx(txHash)
		if err != nil {
			if recoverable {
				return err
			} else {
				return retry.Unrecoverable(err)
			}
		}
		// In a tendermint chain, when the latest height of the chain is N+1,
		// proofs of states updated up to height N are available.
		// In order to make the proof of the state updated by a tx available just after `sendMsgs`,
		// `waitForCommit` must wait until the latest height is greater than the tx height.
		if height, err := c.LatestHeight(); err != nil {
			return fmt.Errorf("failed to obtain latest height: %v", err)
		} else if height.GetRevisionHeight() <= uint64(resTx.Height) {
			return fmt.Errorf("latest_height(%v) is less than or equal to tx_height(%v) yet", height, resTx.Height)
		}
		return nil
	}, retry.Attempts(maxRetry), retry.Delay(retryInterval), rtyErr); err != nil {
		return resTx, fmt.Errorf("failed to make sure that tx is committed: %v", err)
	}

	return resTx, nil
}

// rawQueryTx returns a tx of which hash equals to `hexTxHash`.
// If the RPC is successful but the tx is not found, this returns nil with nil error.
func (c *Chain) rawQueryTx(hexTxHash string) (*coretypes.ResultTx, bool, error) {
	ctx := c.CLIContext(0)

	txHash, err := hex.DecodeString(hexTxHash)
	if err != nil {
		return nil, false, fmt.Errorf("failed to decode the hex string of tx hash: %v", err)
	}

	node, err := ctx.GetNode()
	if err != nil {
		return nil, false, fmt.Errorf("failed to get node: %v", err)
	}

	resTx, err := node.Tx(context.TODO(), txHash, false)
	if err != nil {
		recoverable := !strings.Contains(err.Error(), "transaction indexing is disabled")
		return nil, recoverable, fmt.Errorf("failed to retrieve tx: %v", err)
	}

	return resTx, false, nil
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
	txb, err := txf.BuildUnsignedTx(msgs...)
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

func (c *Chain) SendMsgs(msgs []sdk.Msg) ([]core.MsgID, error) {
	// Broadcast those bytes
	res, err := c.sendMsgs(msgs)
	if err != nil {
		return nil, err
	}
	var msgIDs []core.MsgID
	for msgIndex := range msgs {
		msgIDs = append(msgIDs, &MsgID{
			TxHash:   res.TxHash,
			MsgIndex: uint32(msgIndex),
		})
	}
	return msgIDs, nil
}

func (c *Chain) GetMsgResult(id core.MsgID) (core.MsgResult, error) {
	msgID, ok := id.(*MsgID)
	if !ok {
		return nil, fmt.Errorf("unexpected message id type: %T", id)
	}

	// find tx
	resTx, err := c.waitForCommit(msgID.TxHash)
	if err != nil {
		return nil, fmt.Errorf("failed to query tx: %v", err)
	}

	// check height of the delivered tx
	version := clienttypes.ParseChainID(c.ChainID())
	height := clienttypes.NewHeight(version, uint64(resTx.Height))

	// check if the tx execution succeeded
	if resTx.TxResult.IsErr() {
		err := errors.ABCIError(resTx.TxResult.Codespace, resTx.TxResult.Code, resTx.TxResult.Log)
		txFailureReason := err.Error()
		return &MsgResult{
			height:          height,
			txStatus:        false,
			txFailureReason: txFailureReason,
		}, nil
	}

	// parse the log into ABCI logs
	abciLogs, err := sdk.ParseABCILogs(resTx.TxResult.Log)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ABCI logs: %v", err)
	}

	// parse the ABCI logs into core.MsgEventLog's
	events, err := parseMsgEventLogs(abciLogs, msgID.MsgIndex)
	if err != nil {
		return nil, fmt.Errorf("failed to parse msg event log: %v", err)
	}

	return &MsgResult{
		height:   height,
		txStatus: true,
		events:   events,
	}, nil
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

	return k.Name == name
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
// doesn't cause program to hang.
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
		WithCodec(c.codec).
		WithInterfaceRegistry(c.codec.InterfaceRegistry()).
		WithTxConfig(authtx.NewTxConfig(c.codec, authtx.DefaultSignModes)).
		WithInput(os.Stdin).
		WithNodeURI(c.config.RpcAddr).
		WithClient(c.Client).
		WithAccountRetriever(authTypes.AccountRetriever{}).
		WithBroadcastMode(flags.BroadcastSync).
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

func GetChainLogger() *log.RelayLogger {
	return log.GetLogger().
		WithModule("tendermint.chain")
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
