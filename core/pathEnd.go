package core

import (
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	conntypes "github.com/cosmos/ibc-go/v8/modules/core/03-connection/types"
	chantypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	commitmenttypes "github.com/cosmos/ibc-go/v8/modules/core/23-commitment/types"
)

var (
	DefaultChainPrefix = commitmenttypes.NewMerklePrefix([]byte("ibc"))
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
func (pe *PathEnd) UpdateClient(dstHeader Header, signer sdk.AccAddress) sdk.Msg {
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

func (pe *PathEnd) UpdateClients(dstHeaders []Header, signer sdk.AccAddress) []sdk.Msg {
	var msgs []sdk.Msg
	for _, header := range dstHeaders {
		msgs = append(msgs, pe.UpdateClient(header, signer))
	}
	return msgs
}

// ConnInit creates a MsgConnectionOpenInit
func (pe *PathEnd) ConnInit(dst *PathEnd, signer sdk.AccAddress) sdk.Msg {
	var version *conntypes.Version
	return conntypes.NewMsgConnectionOpenInit(
		pe.ClientID,
		dst.ClientID,
		DefaultChainPrefix,
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
	hostConsensusStateProof []byte,
	signer sdk.AccAddress,
) sdk.Msg {
	cs, err := clienttypes.UnpackClientState(dstClientState.ClientState)
	if err != nil {
		panic(err)
	}
	msg := conntypes.NewMsgConnectionOpenTry(
		pe.ClientID,
		dst.ConnectionID,
		dst.ClientID,
		cs,
		DefaultChainPrefix,
		conntypes.GetCompatibleVersions(),
		DefaultDelayPeriod,
		dstConnState.Proof,
		dstClientState.Proof,
		dstConsState.Proof,
		dstConnState.ProofHeight,
		cs.GetLatestHeight().(clienttypes.Height),
		signer.String(),
	)
	msg.HostConsensusStateProof = hostConsensusStateProof
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
	hostConsensusStateProof []byte,
	signer sdk.AccAddress,
) sdk.Msg {
	cs, err := clienttypes.UnpackClientState(dstClientState.ClientState)
	if err != nil {
		panic(err)
	}
	msg := conntypes.NewMsgConnectionOpenAck(
		pe.ConnectionID,
		dst.ConnectionID,
		cs,
		dstConnState.Proof,
		dstClientState.Proof,
		dstConsState.Proof,
		dstConsState.ProofHeight,
		cs.GetLatestHeight().(clienttypes.Height),
		conntypes.GetCompatibleVersions()[0],
		signer.String(),
	)
	msg.HostConsensusStateProof = hostConsensusStateProof
	if err = msg.ValidateBasic(); err != nil {
		panic(err)
	}
	return msg
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

// ChanUpgradeInit creates a MsgChannelUpgradeInit
func (pe *PathEnd) ChanUpgradeInit(upgradeFields chantypes.UpgradeFields, signer sdk.AccAddress) sdk.Msg {
	return chantypes.NewMsgChannelUpgradeInit(
		pe.PortID,
		pe.ChannelID,
		upgradeFields,
		signer.String(),
	)
}

// ChanUpgradeTry creates a MsgChannelUpgradeTry
func (pe *PathEnd) ChanUpgradeTry(
	newConnectionID string,
	counterpartyChan *chantypes.QueryChannelResponse,
	counterpartyUpg *chantypes.QueryUpgradeResponse,
	signer sdk.AccAddress,
) sdk.Msg {
	return chantypes.NewMsgChannelUpgradeTry(
		pe.PortID,
		pe.ChannelID,
		[]string{newConnectionID},
		counterpartyUpg.Upgrade.Fields,
		counterpartyChan.Channel.UpgradeSequence,
		counterpartyChan.Proof,
		counterpartyUpg.Proof,
		counterpartyChan.ProofHeight,
		signer.String(),
	)
}

// ChanUpgradeAck creates a MsgChannelUpgradeAck
func (pe *PathEnd) ChanUpgradeAck(
	counterpartyChan *chantypes.QueryChannelResponse,
	counterpartyUpg *chantypes.QueryUpgradeResponse,
	signer sdk.AccAddress,
) sdk.Msg {
	return chantypes.NewMsgChannelUpgradeAck(
		pe.PortID,
		pe.ChannelID,
		counterpartyUpg.Upgrade,
		counterpartyChan.Proof,
		counterpartyUpg.Proof,
		counterpartyChan.ProofHeight,
		signer.String(),
	)
}

// ChanUpgradeConfirm creates a MsgChannelUpgradeConfirm
func (pe *PathEnd) ChanUpgradeConfirm(
	counterpartyChan *chantypes.QueryChannelResponse,
	counterpartyUpg *chantypes.QueryUpgradeResponse,
	signer sdk.AccAddress,
) sdk.Msg {
	return chantypes.NewMsgChannelUpgradeConfirm(
		pe.PortID,
		pe.ChannelID,
		counterpartyChan.Channel.State,
		counterpartyUpg.Upgrade,
		counterpartyChan.Proof,
		counterpartyUpg.Proof,
		counterpartyChan.ProofHeight,
		signer.String(),
	)
}

// ChanUpgradeOpen creates a MsgChannelUpgradeOpen
func (pe *PathEnd) ChanUpgradeOpen(
	counterpartyChan *chantypes.QueryChannelResponse,
	signer sdk.AccAddress,
) sdk.Msg {
	return chantypes.NewMsgChannelUpgradeOpen(
		pe.PortID,
		pe.ChannelID,
		counterpartyChan.Channel.State,
		counterpartyChan.Channel.UpgradeSequence,
		counterpartyChan.Proof,
		counterpartyChan.ProofHeight,
		signer.String(),
	)
}

// ChanUpgradeCancel creates a MsgChannelUpgradeCancel
func (pe *PathEnd) ChanUpgradeCancel(
	counterpartyChanUpgErr *chantypes.QueryUpgradeErrorResponse,
	signer sdk.AccAddress,
) sdk.Msg {
	return chantypes.NewMsgChannelUpgradeCancel(
		pe.PortID,
		pe.ChannelID,
		counterpartyChanUpgErr.ErrorReceipt,
		counterpartyChanUpgErr.Proof,
		counterpartyChanUpgErr.ProofHeight,
		signer.String(),
	)
}

// ChanUpgradeTimeout creates a MsgChannelUpgradeTimeout
func (pe *PathEnd) ChanUpgradeTimeout(
	counterpartyChan *chantypes.QueryChannelResponse,
	signer sdk.AccAddress,
) sdk.Msg {
	return chantypes.NewMsgChannelUpgradeTimeout(
		pe.PortID,
		pe.ChannelID,
		*counterpartyChan.Channel,
		counterpartyChan.Proof,
		counterpartyChan.ProofHeight,
		signer.String(),
	)
}

// MsgTransfer creates a new transfer message
func (pe *PathEnd) MsgTransfer(dst *PathEnd, amount sdk.Coin, dstAddr string,
	signer sdk.AccAddress, timeoutHeight, timeoutTimestamp uint64, memo string) sdk.Msg {

	version := clienttypes.ParseChainID(dst.ChainID)
	return transfertypes.NewMsgTransfer(
		pe.PortID,
		pe.ChannelID,
		amount,
		signer.String(),
		dstAddr,
		clienttypes.NewHeight(version, timeoutHeight),
		timeoutTimestamp,
		memo,
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
