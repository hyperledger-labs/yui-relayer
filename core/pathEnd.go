package core

import (
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	transfertypes "github.com/cosmos/ibc-go/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
	commitmenttypes "github.com/cosmos/ibc-go/modules/core/23-commitment/types"
	ibcexported "github.com/cosmos/ibc-go/modules/core/exported"
)

var (
	defaultChainPrefix = commitmenttypes.NewMerklePrefix([]byte("ibc"))
)

const (
	// TODO make it to be configurable
	DefaultDelayPeriod uint64 = 0
)

type PathEndI interface {
	Type() string
	ChainID() string
	ClientID() string
	ConnectionID() string
	ChannelID() string
	PortID() string
	ChannelOrder() channeltypes.Order
	ChannelVersion() string
	Validate() error
	String() string
}

var _ PathEndI = (*PathEnd)(nil)

func (p PathEnd) Type() string {
	return "ibc"
}

func (p PathEnd) ChainID() string {
	return p.ChainId
}

func (p PathEnd) ClientID() string {
	return p.ClientId
}

func (p PathEnd) ConnectionID() string {
	return p.ConnectionId
}

func (p PathEnd) ChannelID() string {
	return p.ChannelId
}

func (p PathEnd) PortID() string {
	return p.PortId
}

func (p PathEnd) ChannelOrder() channeltypes.Order {
	return OrderFromString(strings.ToUpper(p.Order))
}

func (p PathEnd) ChannelVersion() string {
	return p.Version
}

// OrderFromString parses a string into a channel order byte
func OrderFromString(order string) channeltypes.Order {
	switch order {
	case "UNORDERED":
		return channeltypes.UNORDERED
	case "ORDERED":
		return channeltypes.ORDERED
	default:
		return channeltypes.NONE
	}
}

func UpdateClient(pe PathEndI, dstHeader ibcexported.Header, signer sdk.AccAddress) sdk.Msg {
	if err := dstHeader.ValidateBasic(); err != nil {
		panic(err)
	}
	msg, err := clienttypes.NewMsgUpdateClient(
		pe.ClientID(),
		dstHeader,
		signer.String(),
	)
	if err != nil {
		panic(err)
	}
	return msg
}

// ConnInit creates a MsgConnectionOpenInit
func ConnInit(src PathEndI, dst PathEndI, signer sdk.AccAddress) sdk.Msg {
	var version *connectiontypes.Version
	return connectiontypes.NewMsgConnectionOpenInit(
		src.ClientID(),
		dst.ClientID(),
		defaultChainPrefix,
		version,
		DefaultDelayPeriod,
		signer.String(),
	)
}

// ConnTry creates a MsgConnectionOpenTry
// NOTE: ADD NOTE ABOUT PROOF HEIGHT CHANGE HERE
func ConnTry(
	src PathEndI,
	dst PathEndI,
	dstClientState *clienttypes.QueryClientStateResponse,
	dstConnState *connectiontypes.QueryConnectionResponse,
	dstConsState *clienttypes.QueryConsensusStateResponse,
	signer sdk.AccAddress,
) sdk.Msg {
	cs, err := clienttypes.UnpackClientState(dstClientState.ClientState)
	if err != nil {
		panic(err)
	}
	msg := connectiontypes.NewMsgConnectionOpenTry(
		"",
		src.ClientID(),
		dst.ConnectionID(),
		dst.ClientID(),
		cs,
		defaultChainPrefix,
		connectiontypes.ExportedVersionsToProto(connectiontypes.GetCompatibleVersions()),
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
func ConnAck(
	src PathEndI,
	dst PathEndI,
	dstClientState *clienttypes.QueryClientStateResponse,
	dstConnState *connectiontypes.QueryConnectionResponse,
	dstConsState *clienttypes.QueryConsensusStateResponse,
	signer sdk.AccAddress,
) sdk.Msg {
	cs, err := clienttypes.UnpackClientState(dstClientState.ClientState)
	if err != nil {
		panic(err)
	}
	return connectiontypes.NewMsgConnectionOpenAck(
		src.ConnectionID(),
		dst.ConnectionID(),
		cs,
		dstConnState.Proof,
		dstClientState.Proof,
		dstConsState.Proof,
		dstConsState.ProofHeight,
		cs.GetLatestHeight().(clienttypes.Height),
		connectiontypes.ExportedVersionsToProto(connectiontypes.GetCompatibleVersions())[0],
		signer.String(),
	)
}

// ConnConfirm creates a MsgConnectionOpenAck
// NOTE: ADD NOTE ABOUT PROOF HEIGHT CHANGE HERE
func ConnConfirm(pe PathEndI, dstConnState *connectiontypes.QueryConnectionResponse, signer sdk.AccAddress) sdk.Msg {
	return connectiontypes.NewMsgConnectionOpenConfirm(
		pe.ConnectionID(),
		dstConnState.Proof,
		dstConnState.ProofHeight,
		signer.String(),
	)
}

// ChanInit creates a MsgChannelOpenInit
func ChanInit(src PathEndI, dst PathEndI, signer sdk.AccAddress) sdk.Msg {
	return channeltypes.NewMsgChannelOpenInit(
		src.PortID(),
		src.ChannelVersion(),
		src.ChannelOrder(),
		[]string{src.ConnectionID()},
		dst.PortID(),
		signer.String(),
	)
}

// ChanTry creates a MsgChannelOpenTry
func ChanTry(src PathEndI, dst PathEndI, dstChanState *channeltypes.QueryChannelResponse, signer sdk.AccAddress) sdk.Msg {
	return channeltypes.NewMsgChannelOpenTry(
		src.PortID(),
		"",
		src.ChannelVersion(),
		dstChanState.Channel.Ordering,
		[]string{src.ConnectionID()},
		dst.PortID(),
		dst.ChannelID(),
		dstChanState.Channel.Version,
		dstChanState.Proof,
		dstChanState.ProofHeight,
		signer.String(),
	)
}

// ChanAck creates a MsgChannelOpenAck
func ChanAck(src PathEndI, dst PathEndI, dstChanState *channeltypes.QueryChannelResponse, signer sdk.AccAddress) sdk.Msg {
	return channeltypes.NewMsgChannelOpenAck(
		src.PortID(),
		src.ChannelID(),
		dst.ChannelID(),
		dstChanState.Channel.Version,
		dstChanState.Proof,
		dstChanState.ProofHeight,
		signer.String(),
	)
}

// ChanConfirm creates a MsgChannelOpenConfirm
func ChanConfirm(pe PathEndI, dstChanState *channeltypes.QueryChannelResponse, signer sdk.AccAddress) sdk.Msg {
	return channeltypes.NewMsgChannelOpenConfirm(
		pe.PortID(),
		pe.ChannelID(),
		dstChanState.Proof,
		dstChanState.ProofHeight,
		signer.String(),
	)
}

// ChanCloseInit creates a MsgChannelCloseInit
func ChanCloseInit(pe PathEndI, signer sdk.AccAddress) sdk.Msg {
	return channeltypes.NewMsgChannelCloseInit(
		pe.PortID(),
		pe.ChannelID(),
		signer.String(),
	)
}

// ChanCloseConfirm creates a MsgChannelCloseConfirm
func ChanCloseConfirm(pe PathEndI, dstChanState *channeltypes.QueryChannelResponse, signer sdk.AccAddress) sdk.Msg {
	return channeltypes.NewMsgChannelCloseConfirm(
		pe.PortID(),
		pe.ChannelID(),
		dstChanState.Proof,
		dstChanState.ProofHeight,
		signer.String(),
	)
}

// MsgTransfer creates a new transfer message
func MsgTransfer(src PathEndI, dst PathEndI, amount sdk.Coin, dstAddr string,
	signer sdk.AccAddress, timeoutHeight, timeoutTimestamp uint64) sdk.Msg {

	version := clienttypes.ParseChainID(dst.ChainID())
	return transfertypes.NewMsgTransfer(
		src.PortID(),
		src.ChannelID(),
		amount,
		signer.String(),
		dstAddr,
		clienttypes.NewHeight(version, timeoutHeight),
		timeoutTimestamp,
	)
}
