package fabric

import (
	"fmt"
	"path/filepath"
	"strings"

	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	conntypes "github.com/cosmos/ibc-go/modules/core/03-connection/types"
	chantypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
	ibcexported "github.com/cosmos/ibc-go/modules/core/exported"
	"github.com/gogo/protobuf/proto"
	fabrictypes "github.com/hyperledger-labs/yui-fabric-ibc/x/ibc/light-clients/xx-fabric/types"
	"github.com/hyperledger-labs/yui-relayer/core"
	msppb "github.com/hyperledger/fabric-protos-go/msp"
	"github.com/hyperledger/fabric-sdk-go/pkg/gateway"
	"github.com/hyperledger/fabric/bccsp/factory"
	"github.com/hyperledger/fabric/msp"
)

type Prover struct {
	chain  *Chain
	config ProverConfig
}

var _ core.ProverI = (*Prover)(nil)

func NewProver(chain *Chain, config ProverConfig) *Prover {
	return &Prover{chain: chain, config: config}
}

// GetChainID returns the chain ID
func (pr *Prover) GetChainID() string {
	return pr.chain.ChainID()
}

// QueryClientStateWithProof returns the ClientState and its proof
func (pr *Prover) QueryClientStateWithProof(_ int64) (*clienttypes.QueryClientStateResponse, error) {
	cs, proof, err := pr.endorseClientState(pr.chain.pathEnd.ClientID)
	if err != nil {
		return nil, err
	}
	anyCS, err := clienttypes.PackClientState(cs)
	if err != nil {
		return nil, err
	}
	proofBytes, err := proto.Marshal(proof)
	if err != nil {
		return nil, err
	}
	return &clienttypes.QueryClientStateResponse{
		ClientState: anyCS,
		Proof:       proofBytes,
		ProofHeight: pr.getCurrentHeight(),
	}, nil
}

// QueryClientConsensusState returns the ClientConsensusState and its proof
func (pr *Prover) QueryClientConsensusStateWithProof(_ int64, dstClientConsHeight ibcexported.Height) (*clienttypes.QueryConsensusStateResponse, error) {
	css, proof, err := pr.endorseConsensusState(pr.chain.Path().ClientID, dstClientConsHeight.GetRevisionHeight())
	if err != nil {
		return nil, err
	}
	anyCSS, err := clienttypes.PackConsensusState(css)
	if err != nil {
		return nil, err
	}
	proofBytes, err := proto.Marshal(proof)
	if err != nil {
		return nil, err
	}
	return &clienttypes.QueryConsensusStateResponse{
		ConsensusState: anyCSS,
		Proof:          proofBytes,
		ProofHeight:    pr.getCurrentHeight(),
	}, nil
}

// QueryConnectionWithProof returns the Connection and its proof
func (pr *Prover) QueryConnectionWithProof(_ int64) (*conntypes.QueryConnectionResponse, error) {
	conn, proof, err := pr.endorseConnectionState(pr.chain.pathEnd.ConnectionID)
	if err != nil {
		if strings.Contains(err.Error(), conntypes.ErrConnectionNotFound.Error()) {
			return emptyConnRes, nil
		} else {
			return nil, err
		}
	}
	proofBytes, err := proto.Marshal(proof)
	if err != nil {
		return nil, err
	}
	return &conntypes.QueryConnectionResponse{
		Connection:  conn,
		Proof:       proofBytes,
		ProofHeight: pr.getCurrentHeight(),
	}, nil
}

var emptyChannelRes = chantypes.NewQueryChannelResponse(
	chantypes.NewChannel(
		chantypes.UNINITIALIZED,
		chantypes.UNORDERED,
		chantypes.NewCounterparty(
			"port",
			"channel",
		),
		[]string{},
		"version",
	),
	[]byte{},
	clienttypes.NewHeight(0, 0),
)

// QueryChannelWithProof returns the Channel and its proof
func (pr *Prover) QueryChannelWithProof(_ int64) (chanRes *chantypes.QueryChannelResponse, err error) {
	channel, proof, err := pr.endorseChannelState(pr.chain.pathEnd.PortID, pr.chain.pathEnd.ChannelID)
	if err != nil {
		if strings.Contains(err.Error(), chantypes.ErrChannelNotFound.Error()) {
			return emptyChannelRes, nil
		} else {
			return nil, err
		}
	}
	proofBytes, err := proto.Marshal(proof)
	if err != nil {
		return nil, err
	}
	return &chantypes.QueryChannelResponse{
		Channel:     channel,
		Proof:       proofBytes,
		ProofHeight: pr.getCurrentHeight(),
	}, nil
}

// QueryPacketCommitmentWithProof returns the packet commitment and its proof
func (pr *Prover) QueryPacketCommitmentWithProof(_ int64, seq uint64) (comRes *chantypes.QueryPacketCommitmentResponse, err error) {
	cm, proof, err := pr.endorsePacketCommitment(pr.chain.Path().PortID, pr.chain.Path().ChannelID, seq)
	if err != nil {
		return nil, err
	}
	proofBytes, err := proto.Marshal(proof)
	if err != nil {
		return nil, err
	}
	return &chantypes.QueryPacketCommitmentResponse{
		Commitment:  cm,
		Proof:       proofBytes,
		ProofHeight: pr.getCurrentHeight(),
	}, nil
}

// QueryPacketAcknowledgementCommitmentWithProof returns the packet acknowledgement commitment and its proof
func (pr *Prover) QueryPacketAcknowledgementCommitmentWithProof(_ int64, seq uint64) (ackRes *chantypes.QueryPacketAcknowledgementResponse, err error) {
	cm, proof, err := pr.endorsePacketAcknowledgement(pr.chain.Path().PortID, pr.chain.Path().ChannelID, seq)
	if err != nil {
		return nil, err
	}
	proofBytes, err := proto.Marshal(proof)
	if err != nil {
		return nil, err
	}
	return &chantypes.QueryPacketAcknowledgementResponse{
		Acknowledgement: cm,
		Proof:           proofBytes,
		ProofHeight:     pr.getCurrentHeight(),
	}, nil
}

// QueryLatestHeader returns the latest header from the chain
func (pr *Prover) QueryLatestHeader() (out core.HeaderI, err error) {
	seq, err := pr.chain.QueryCurrentSequence()
	if err != nil {
		return nil, err
	}

	var ccid = fabrictypes.ChaincodeID{
		Name:    pr.chain.config.ChaincodeId,
		Version: "1", // TODO add version to config
	}

	pcBytes, err := makeEndorsementPolicy(pr.config.EndorsementPolicies)
	if err != nil {
		return nil, err
	}
	ipBytes, err := makeIBCPolicy(pr.config.IbcPolicies)
	if err != nil {
		return nil, err
	}
	ci := fabrictypes.NewChaincodeInfo(pr.chain.config.Channel, ccid, pcBytes, ipBytes, nil)
	ch := fabrictypes.NewChaincodeHeader(
		seq.Value,
		seq.Timestamp,
		fabrictypes.CommitmentProof{},
	)
	mspConfs, err := pr.GetLocalMspConfigs()
	if err != nil {
		return nil, err
	}
	hs := []fabrictypes.MSPHeader{}
	for _, mc := range mspConfs {
		mcBytes, err := proto.Marshal(&mc.Config)
		if err != nil {
			return nil, err
		}
		hs = append(hs, fabrictypes.NewMSPHeader(fabrictypes.MSPHeaderTypeCreate, mc.MSPID, mcBytes, ipBytes, &fabrictypes.MessageProof{}))
	}
	mhs := fabrictypes.NewMSPHeaders(hs)
	header := fabrictypes.NewHeader(&ch, &ci, &mhs)

	if err := header.ValidateBasic(); err != nil {
		return nil, err
	}

	return header, nil
}

// GetLatestLightHeight uses the CLI utilities to pull the latest height from a given chain
// Fabric prover doesn't support "chain" height
func (pr *Prover) GetLatestLightHeight() (int64, error) {
	return -1, nil
}

// SetupHeader creates a new header based on a given header
func (*Prover) SetupHeader(dst core.LightClientIBCQueryierI, baseSrcHeader core.HeaderI) (core.HeaderI, error) {
	return nil, nil
}

// UpdateLightWithHeader returns the light header and its height
func (pr *Prover) UpdateLightWithHeader() (header core.HeaderI, provableHeight int64, queryableHeight int64, err error) {
	h, err := pr.QueryLatestHeader()
	if err != nil {
		return nil, -1, -1, err
	}
	return h, -1, -1, nil
}

func (prv *Prover) getCurrentHeight() clienttypes.Height {
	seq, err := prv.chain.QueryCurrentSequence()
	if err != nil {
		panic(err)
	}
	return clienttypes.NewHeight(0, seq.Value)
}

type MSPConfig struct {
	MSPID  string
	Config msppb.MSPConfig
}

// get MSP Configs for Chain.IBCPolicies
func (pr *Prover) GetLocalMspConfigs() ([]MSPConfig, error) {
	if len(pr.config.IbcPolicies) != len(pr.config.MspConfigPaths) {
		return nil, fmt.Errorf("IBCPolicies and MspConfigPaths must have the same length for now: %v != %v", len(pr.config.IbcPolicies), len(pr.config.MspConfigPaths))
	}
	res := []MSPConfig{}
	for i, path := range pr.config.MspConfigPaths {
		mspId := pr.config.IbcPolicies[i]
		bccspConfig := factory.GetDefaultOpts()
		mspConf, err := msp.GetLocalMspConfig(filepath.Clean(path), bccspConfig, mspId)
		if err != nil {
			return nil, err
		}
		if err := getVerifyingConfig(mspConf); err != nil {
			return nil, err
		}
		res = append(res, MSPConfig{MSPID: mspId, Config: *mspConf})
	}
	return res, nil
}

func (c *Chain) getSerializedIdentity(label string) (*msppb.SerializedIdentity, error) {
	creds, err := c.gateway.Wallet.Get(label)
	if err != nil {
		return nil, err
	}
	identity := creds.(*gateway.X509Identity)
	return &msppb.SerializedIdentity{
		Mspid:   identity.MspID,
		IdBytes: []byte(identity.Certificate()),
	}, nil
}

// remove SigningIdentity for verifying only purpose.
func getVerifyingConfig(mconf *msppb.MSPConfig) error {
	var conf msppb.FabricMSPConfig
	err := proto.Unmarshal(mconf.Config, &conf)
	if err != nil {
		return err
	}
	conf.SigningIdentity = nil
	confb, err := proto.Marshal(&conf)
	if err != nil {
		return err
	}
	mconf.Config = confb
	return nil
}
