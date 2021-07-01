package core

import (
	"strings"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	transfertypes "github.com/cosmos/ibc-go/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	conntypes "github.com/cosmos/ibc-go/modules/core/03-connection/types"
	chantypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
	commitmenttypes "github.com/cosmos/ibc-go/modules/core/23-commitment/types"
	commitmentypes "github.com/cosmos/ibc-go/modules/core/23-commitment/types"
	ibcexported "github.com/cosmos/ibc-go/modules/core/exported"
	tmclient "github.com/cosmos/ibc-go/modules/light-clients/07-tendermint/types"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/light"
)

var (
	defaultChainPrefix = commitmentypes.NewMerklePrefix([]byte("ibc"))
)

const (
	// TODO make it to be configurable
	DefaultDelayPeriod uint64 = 0
)

// PathEnd represents the local connection identifers for a relay path
// The path is set on the chain before performing operations
type PathEnd struct {
	ChainID      string `yaml:"chain-id,omitempty" json:"chain-id,omitempty"`
	ClientID     string `yaml:"client-id,omitempty" json:"client-id,omitempty"`
	ConnectionID string `yaml:"connection-id,omitempty" json:"connection-id,omitempty"`
	ChannelID    string `yaml:"channel-id,omitempty" json:"channel-id,omitempty"`
	PortID       string `yaml:"port-id,omitempty" json:"port-id,omitempty"`
	Order        string `yaml:"order,omitempty" json:"order,omitempty"`
	Version      string `yaml:"version,omitempty" json:"version,omitempty"`
}

// OrderFromString parses a string into a channel order byte
func OrderFromString(order string) chantypes.Order {
	switch order {
	case "UNORDERED":
		return chantypes.UNORDERED
	case "ORDERED":
		return chantypes.ORDERED
	default:
		return chantypes.NONE
	}
}

func (pe *PathEnd) GetOrder() chantypes.Order {
	return OrderFromString(strings.ToUpper(pe.Order))
}

// UpdateClient creates an sdk.Msg to update the client on src with data pulled from dst
func (pe *PathEnd) UpdateClient(dstHeader ibcexported.Header, signer sdk.AccAddress) sdk.Msg {
	if err := dstHeader.ValidateBasic(); err != nil {
		panic(err)
	}
	msg, err := clienttypes.NewMsgUpdateClient(
		pe.ClientID,
		dstHeader,
		signer.String(),
	)
	if err != nil {
		panic(err)
	}
	return msg
}

// CreateClient creates an sdk.Msg to update the client on src with consensus state from dst
func (pe *PathEnd) CreateClient(
	dstHeader *tmclient.Header,
	trustingPeriod, unbondingPeriod time.Duration,
	consensusParams *abci.ConsensusParams,
	signer sdk.AccAddress) sdk.Msg {
	if err := dstHeader.ValidateBasic(); err != nil {
		panic(err)
	}

	// Blank Client State
	clientState := tmclient.NewClientState(
		dstHeader.GetHeader().GetChainID(),
		tmclient.NewFractionFromTm(light.DefaultTrustLevel),
		trustingPeriod,
		unbondingPeriod,
		time.Minute*10,
		dstHeader.GetHeight().(clienttypes.Height),
		commitmenttypes.GetSDKSpecs(),
		[]string{"upgrade", "upgradedIBCState"},
		false,
		false,
	)

	msg, err := clienttypes.NewMsgCreateClient(
		clientState,
		dstHeader.ConsensusState(),
		signer.String(),
	)

	if err != nil {
		panic(err)
	}
	if err = msg.ValidateBasic(); err != nil {
		panic(err)
	}
	return msg
}

// ConnInit creates a MsgConnectionOpenInit
func (pe *PathEnd) ConnInit(dst *PathEnd, signer sdk.AccAddress) sdk.Msg {
	var version *conntypes.Version
	return conntypes.NewMsgConnectionOpenInit(
		pe.ClientID,
		dst.ClientID,
		defaultChainPrefix,
		version,
		DefaultDelayPeriod,
		signer.String(),
	)
}

// ConnTry creates a MsgConnectionOpenTry
// NOTE: ADD NOTE ABOUT PROOF HEIGHT CHANGE HERE
func (pe *PathEnd) ConnTry(
	dst *PathEnd,
	dstClientState *clienttypes.QueryClientStateResponse,
	dstConnState *conntypes.QueryConnectionResponse,
	dstConsState *clienttypes.QueryConsensusStateResponse,
	signer sdk.AccAddress,
) sdk.Msg {
	cs, err := clienttypes.UnpackClientState(dstClientState.ClientState)
	if err != nil {
		panic(err)
	}
	msg := conntypes.NewMsgConnectionOpenTry(
		"",
		pe.ClientID,
		dst.ConnectionID,
		dst.ClientID,
		cs,
		defaultChainPrefix,
		conntypes.ExportedVersionsToProto(conntypes.GetCompatibleVersions()),
		DefaultDelayPeriod,
		dstConnState.Proof,
		dstClientState.Proof,
		dstConsState.Proof,
		dstConnState.ProofHeight,
		cs.GetLatestHeight().(clienttypes.Height),
		signer.String(),
	)
	if err = msg.ValidateBasic(); err != nil {
		panic(err)
	}
	return msg
}

// ConnAck creates a MsgConnectionOpenAck
// NOTE: ADD NOTE ABOUT PROOF HEIGHT CHANGE HERE
func (pe *PathEnd) ConnAck(
	dst *PathEnd,
	dstClientState *clienttypes.QueryClientStateResponse,
	dstConnState *conntypes.QueryConnectionResponse,
	dstConsState *clienttypes.QueryConsensusStateResponse,
	signer sdk.AccAddress,
) sdk.Msg {
	cs, err := clienttypes.UnpackClientState(dstClientState.ClientState)
	if err != nil {
		panic(err)
	}
	return conntypes.NewMsgConnectionOpenAck(
		pe.ConnectionID,
		dst.ConnectionID,
		cs,
		dstConnState.Proof,
		dstClientState.Proof,
		dstConsState.Proof,
		dstConsState.ProofHeight,
		cs.GetLatestHeight().(clienttypes.Height),
		conntypes.ExportedVersionsToProto(conntypes.GetCompatibleVersions())[0],
		signer.String(),
	)
}

// ConnConfirm creates a MsgConnectionOpenAck
// NOTE: ADD NOTE ABOUT PROOF HEIGHT CHANGE HERE
func (pe *PathEnd) ConnConfirm(dstConnState *conntypes.QueryConnectionResponse, signer sdk.AccAddress) sdk.Msg {
	return conntypes.NewMsgConnectionOpenConfirm(
		pe.ConnectionID,
		dstConnState.Proof,
		dstConnState.ProofHeight,
		signer.String(),
	)
}

// ChanInit creates a MsgChannelOpenInit
func (pe *PathEnd) ChanInit(dst *PathEnd, signer sdk.AccAddress) sdk.Msg {
	return chantypes.NewMsgChannelOpenInit(
		pe.PortID,
		pe.Version,
		pe.GetOrder(),
		[]string{pe.ConnectionID},
		dst.PortID,
		signer.String(),
	)
}

// ChanTry creates a MsgChannelOpenTry
func (pe *PathEnd) ChanTry(dst *PathEnd, dstChanState *chantypes.QueryChannelResponse, signer sdk.AccAddress) sdk.Msg {
	return chantypes.NewMsgChannelOpenTry(
		pe.PortID,
		"",
		pe.Version,
		dstChanState.Channel.Ordering,
		[]string{pe.ConnectionID},
		dst.PortID,
		dst.ChannelID,
		dstChanState.Channel.Version,
		dstChanState.Proof,
		dstChanState.ProofHeight,
		signer.String(),
	)
}

// ChanAck creates a MsgChannelOpenAck
func (pe *PathEnd) ChanAck(dst *PathEnd, dstChanState *chantypes.QueryChannelResponse, signer sdk.AccAddress) sdk.Msg {
	return chantypes.NewMsgChannelOpenAck(
		pe.PortID,
		pe.ChannelID,
		dst.ChannelID,
		dstChanState.Channel.Version,
		dstChanState.Proof,
		dstChanState.ProofHeight,
		signer.String(),
	)
}

// ChanConfirm creates a MsgChannelOpenConfirm
func (pe *PathEnd) ChanConfirm(dstChanState *chantypes.QueryChannelResponse, signer sdk.AccAddress) sdk.Msg {
	return chantypes.NewMsgChannelOpenConfirm(
		pe.PortID,
		pe.ChannelID,
		dstChanState.Proof,
		dstChanState.ProofHeight,
		signer.String(),
	)
}

// ChanCloseInit creates a MsgChannelCloseInit
func (pe *PathEnd) ChanCloseInit(signer sdk.AccAddress) sdk.Msg {
	return chantypes.NewMsgChannelCloseInit(
		pe.PortID,
		pe.ChannelID,
		signer.String(),
	)
}

// ChanCloseConfirm creates a MsgChannelCloseConfirm
func (pe *PathEnd) ChanCloseConfirm(dstChanState *chantypes.QueryChannelResponse, signer sdk.AccAddress) sdk.Msg {
	return chantypes.NewMsgChannelCloseConfirm(
		pe.PortID,
		pe.ChannelID,
		dstChanState.Proof,
		dstChanState.ProofHeight,
		signer.String(),
	)
}

// MsgTransfer creates a new transfer message
func (pe *PathEnd) MsgTransfer(dst *PathEnd, amount sdk.Coin, dstAddr string,
	signer sdk.AccAddress, timeoutHeight, timeoutTimestamp uint64) sdk.Msg {

	version := clienttypes.ParseChainID(dst.ChainID)
	return transfertypes.NewMsgTransfer(
		pe.PortID,
		pe.ChannelID,
		amount,
		signer.String(),
		dstAddr,
		clienttypes.NewHeight(version, timeoutHeight),
		timeoutTimestamp,
	)
}

// NewPacket returns a new packet from src to dist w
func (pe *PathEnd) NewPacket(dst *PathEnd, sequence uint64, packetData []byte,
	timeoutHeight, timeoutStamp uint64) chantypes.Packet {
	version := clienttypes.ParseChainID(dst.ChainID)
	return chantypes.NewPacket(
		packetData,
		sequence,
		pe.PortID,
		pe.ChannelID,
		dst.PortID,
		dst.ChannelID,
		clienttypes.NewHeight(version, timeoutHeight),
		timeoutStamp,
	)
}

// XferPacket creates a new transfer packet
func (pe *PathEnd) XferPacket(amount sdk.Coin, sender, receiver string) []byte {
	return transfertypes.NewFungibleTokenPacketData(
		amount.Denom,
		amount.Amount.Uint64(),
		sender,
		receiver,
	).GetBytes()
}
