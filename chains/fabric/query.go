package fabric

import (
	"encoding/base64"
	"encoding/json"

	clienttypes "github.com/cosmos/cosmos-sdk/x/ibc/core/02-client/types"
	"github.com/datachainlab/fabric-ibc/app"
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
func (c *Chain) QueryClientState(height int64) (*clienttypes.QueryClientStateResponse, error) {
	req := &clienttypes.QueryClientStateRequest{
		ClientId: c.pathEnd.ClientID,
	}
	bz, err := proto.Marshal(req)
	if err != nil {
		return nil, err
	}
	res, err := c.Query(app.RequestQuery{
		Data: string(bz),
		Path: "/ibc.core.client.v1.Query/ClientState",
	})
	if err != nil {
		return nil, err
	}
	bz, err = base64.StdEncoding.DecodeString(res.Value)
	if err != nil {
		return nil, err
	}
	var cres clienttypes.QueryClientStateResponse
	if err := cres.Unmarshal(bz); err != nil {
		return nil, err
	}
	return &cres, nil
}

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
