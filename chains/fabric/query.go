package fabric

import (
	"encoding/json"
	"fmt"
	"strings"

	clienttypes "github.com/cosmos/cosmos-sdk/x/ibc/core/02-client/types"
	conntypes "github.com/cosmos/cosmos-sdk/x/ibc/core/03-connection/types"
	committypes "github.com/cosmos/cosmos-sdk/x/ibc/core/23-commitment/types"
	ibcexported "github.com/cosmos/cosmos-sdk/x/ibc/core/exported"
	"github.com/datachainlab/fabric-ibc/app"
	"github.com/datachainlab/fabric-ibc/chaincode"
	"github.com/datachainlab/fabric-ibc/commitment"
	fabrictypes "github.com/datachainlab/fabric-ibc/x/ibc/light-clients/xx-fabric/types"
	"github.com/datachainlab/relayer/core"
	"github.com/gogo/protobuf/proto"
)

const (
	queryFunc       = "query"
	getSequenceFunc = "getSequence"
)

func (c *Chain) Query(req app.RequestQuery) (*app.ResponseQuery, error) {
	bz, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	r, err := c.Contract().EvaluateTransaction(queryFunc, string(bz))
	if err != nil {
		return nil, err
	}
	var res app.ResponseQuery
	if err := json.Unmarshal(r, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// QueryClientState retrevies the latest consensus state for a client in state at a given height
func (c *Chain) QueryClientState(height int64, prove bool) (*clienttypes.QueryClientStateResponse, error) {
	if prove {
		return c.queryClientStateWithProof(c.pathEnd.ClientID)
	}

	req := &clienttypes.QueryClientStateRequest{
		ClientId: c.pathEnd.ClientID,
	}
	var cres clienttypes.QueryClientStateResponse
	if err := c.query("/ibc.core.client.v1.Query/ClientState", req, &cres); err != nil {
		return nil, err
	}
	return &cres, nil
}

func (c *Chain) queryClientStateWithProof(clientID string) (*clienttypes.QueryClientStateResponse, error) {
	cs, proof, err := c.endorseClientState(clientID)
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
		ProofHeight: c.getCurrentHeight(),
	}, nil
}

func (c *Chain) QueryClientConsensusState(height int64, dstClientConsHeight ibcexported.Height, prove bool) (*clienttypes.QueryConsensusStateResponse, error) {
	if prove {
		return c.queryClientConsensusStateWithProof(dstClientConsHeight)
	}
	fmt.Println("Try to QueryClientConsensusState:", height, dstClientConsHeight.String())
	req := &clienttypes.QueryConsensusStateRequest{
		ClientId:      c.Path().ClientID,
		VersionNumber: dstClientConsHeight.GetVersionNumber(),
		VersionHeight: dstClientConsHeight.GetVersionHeight(),
	}
	var cres clienttypes.QueryConsensusStateResponse
	if err := c.query("/ibc.core.client.v1.Query/ConsensusState", req, &cres); err != nil {
		return nil, err
	}
	return &cres, nil
}

func (c *Chain) queryClientConsensusStateWithProof(height ibcexported.Height) (*clienttypes.QueryConsensusStateResponse, error) {
	css, proof, err := c.endorseConsensusState(c.Path().ClientID, height.GetVersionHeight())
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
		ProofHeight:    c.getCurrentHeight(),
	}, nil
}

// QueryConnection returns the remote end of a given connection
func (c *Chain) QueryConnection(height int64, prove bool) (*conntypes.QueryConnectionResponse, error) {
	if prove {
		if res, err := c.queryConnectioWithProof(c.pathEnd.ConnectionID); err == nil {
			return res, nil
		} else if strings.Contains(err.Error(), "connection not found") {
			return emptyConnRes, nil
		} else {
			return nil, err
		}
	}
	req := &conntypes.QueryConnectionRequest{
		ConnectionId: c.pathEnd.ConnectionID,
	}
	var cres conntypes.QueryConnectionResponse
	if err := c.query("/ibc.core.connection.v1.Query/Connection", req, &cres); err != nil {
		if strings.Contains(err.Error(), "connection not found: key not found") {
			return emptyConnRes, nil
		}
		return nil, err
	}
	return &cres, nil
}

func (c *Chain) queryConnectioWithProof(connectionID string) (*conntypes.QueryConnectionResponse, error) {
	conn, proof, err := c.endorseConnectionState(connectionID)
	if err != nil {
		return nil, err
	}
	proofBytes, err := proto.Marshal(proof)
	if err != nil {
		return nil, err
	}
	return &conntypes.QueryConnectionResponse{
		Connection:  conn,
		Proof:       proofBytes,
		ProofHeight: c.getCurrentHeight(),
	}, nil
}

var emptyConnRes = conntypes.NewQueryConnectionResponse(
	conntypes.NewConnectionEnd(
		conntypes.UNINITIALIZED,
		"client",
		conntypes.NewCounterparty(
			"client",
			"connection",
			committypes.NewMerklePrefix([]byte{}),
		),
		[]*conntypes.Version{},
	),
	[]byte{},
	clienttypes.NewHeight(0, 0),
)

func (c *Chain) QueryLatestHeight() (int64, error) {
	seq, err := c.QueryCurrentSequence()
	if err != nil {
		return 0, err
	}
	return int64(seq.GetValue()), nil
}

// QueryCurrentSequence returns the current sequence for IBC chaincode
func (c *Chain) QueryCurrentSequence() (*commitment.Sequence, error) {
	res, err := c.Contract().EvaluateTransaction(getSequenceFunc)
	if err != nil {
		return nil, err
	}
	seq := new(commitment.Sequence)
	if err := json.Unmarshal(res, seq); err != nil {
		return nil, err
	}
	return seq, nil
}

func (c *Chain) QueryLatestHeader() (core.HeaderI, error) {
	seq, err := c.QueryCurrentSequence()
	if err != nil {
		return nil, err
	}

	var ccid = fabrictypes.ChaincodeID{
		Name:    c.config.ChaincodeId,
		Version: "1", // TODO add version to config
	}

	pcBytes, err := makeEndorsementPolicy(c.config.EndorsementPolicies)
	if err != nil {
		return nil, err
	}
	ipBytes, err := makeIBCPolicy(c.config.IbcPolicies)
	if err != nil {
		return nil, err
	}
	ci := fabrictypes.NewChaincodeInfo(c.config.Channel, ccid, pcBytes, ipBytes, nil)
	ch := fabrictypes.NewChaincodeHeader(
		seq.Value,
		seq.Timestamp,
		fabrictypes.CommitmentProof{},
	)
	mspConfs, err := c.GetLocalMspConfigs()
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

func (c *Chain) query(path string, req proto.Message, res interface{ Unmarshal(bz []byte) error }) error {
	bz, err := proto.Marshal(req)
	if err != nil {
		return err
	}
	r, err := c.Query(app.RequestQuery{
		Data: chaincode.EncodeToString(bz),
		Path: path,
	})
	if err != nil {
		return err
	}
	bz, err = chaincode.DecodeString(r.Value)
	if err != nil {
		return err
	}
	return res.Unmarshal(bz)
}

func (c *Chain) getCurrentHeight() clienttypes.Height {
	seq, err := c.QueryCurrentSequence()
	if err != nil {
		panic(err)
	}
	return clienttypes.NewHeight(0, seq.Value)
}
