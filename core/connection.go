package core

import (
	"fmt"
	"log"
	"time"

	retry "github.com/avast/retry-go"
	sdk "github.com/cosmos/cosmos-sdk/types"
	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	conntypes "github.com/cosmos/ibc-go/modules/core/03-connection/types"
	ibcexported "github.com/cosmos/ibc-go/modules/core/exported"
)

var (
	rtyAttNum = uint(5)
	rtyAtt    = retry.Attempts(rtyAttNum)
	rtyDel    = retry.Delay(time.Millisecond * 400)
	rtyErr    = retry.LastErrorOnly(true)
)

func CreateConnection(src, dst *ProvableChain, to time.Duration) error {
	ticker := time.NewTicker(to)

	failed := 0
	for ; true; <-ticker.C {
		connSteps, err := createConnectionStep(src, dst)
		if err != nil {
			return err
		}

		if !connSteps.Ready() {
			break
		}

		connSteps.Send(src, dst)

		switch {
		// In the case of success and this being the last transaction
		// debug logging, log created connection and break
		case connSteps.Success() && connSteps.Last:
			log.Println(fmt.Sprintf("★ Connection created: [%s]client{%s}conn{%s} -> [%s]client{%s}conn{%s}",
				src.ChainID(), src.Path().ClientID, src.Path().ConnectionID,
				dst.ChainID(), dst.Path().ClientID, dst.Path().ConnectionID))
			return nil
		// In the case of success, reset the failures counter
		case connSteps.Success():
			failed = 0
			continue
		// In the case of failure, increment the failures counter and exit if this is the 3rd failure
		case !connSteps.Success():
			failed++
			log.Println(fmt.Sprintf("retrying transaction..."))
			time.Sleep(5 * time.Second)
			if failed > 2 {
				return fmt.Errorf("! Connection failed: [%s]client{%s}conn{%s} -> [%s]client{%s}conn{%s}",
					src.ChainID(), src.Path().ClientID, src.Path().ConnectionID,
					dst.ChainID(), dst.Path().ClientID, dst.Path().ConnectionID)
			}
		}

	}

	return nil
}

func createConnectionStep(src, dst *ProvableChain) (*RelayMsgs, error) {
	out := NewRelayMsgs()
	if err := validatePaths(src, dst); err != nil {
		return nil, err
	}
	// First, update the light clients to the latest header and return the header
	sh, err := NewSyncHeaders(src, dst)
	if err != nil {
		return nil, err
	}
	// Query a number of things all at once
	var (
		srcUpdateHeader, dstUpdateHeader HeaderI
		srcCsRes, dstCsRes               *clienttypes.QueryClientStateResponse
		srcCS, dstCS                     ibcexported.ClientState
		srcCons, dstCons                 *clienttypes.QueryConsensusStateResponse
		srcConsH, dstConsH               ibcexported.Height
	)
	err = retry.Do(func() error {
		srcUpdateHeader, dstUpdateHeader, err = sh.GetHeaders(src, dst)
		return err
	}, rtyAtt, rtyDel, rtyErr, retry.OnRetry(func(n uint, err error) {
		// logRetryUpdateHeaders(src, dst, n, err)
		if err := sh.Updates(src, dst); err != nil {
			panic(err)
		}
	}))
	if err != nil {
		return nil, err
	}

	srcConn, dstConn, err := QueryConnectionPair(src, dst, sh.GetProvableHeight(src.ChainID()), sh.GetProvableHeight(dst.ChainID()))
	if err != nil {
		return nil, err
	}

	if !(srcConn.Connection.State == conntypes.UNINITIALIZED && dstConn.Connection.State == conntypes.UNINITIALIZED) {
		// Query client state from each chain's client
		srcCsRes, dstCsRes, err = QueryClientStatePair(src, dst, sh.GetProvableHeight(src.ChainID()), sh.GetProvableHeight(dst.ChainID()))
		if err != nil && (srcCsRes == nil || dstCsRes == nil) {
			return nil, err
		}
		if err := src.Codec().UnpackAny(srcCsRes.ClientState, &srcCS); err != nil {
			return nil, err
		}
		if err := dst.Codec().UnpackAny(dstCsRes.ClientState, &dstCS); err != nil {
			return nil, err
		}

		// Store the heights
		srcConsH, dstConsH = srcCS.GetLatestHeight(), dstCS.GetLatestHeight()
		srcCons, dstCons, err = QueryClientConsensusStatePair(
			src, dst, sh.GetProvableHeight(src.ChainID()), sh.GetProvableHeight(dst.ChainID()), srcConsH, dstConsH)
		if err != nil {
			return nil, err
		}
	}

	switch {
	// Handshake hasn't been started on src or dst, relay `connOpenInit` to src
	case srcConn.Connection.State == conntypes.UNINITIALIZED && dstConn.Connection.State == conntypes.UNINITIALIZED:
		logConnectionStates(src, dst, srcConn, dstConn)
		addr := mustGetAddress(src)
		if dstUpdateHeader != nil {
			out.Src = append(out.Src, src.Path().UpdateClient(dstUpdateHeader, addr))
		}
		out.Src = append(out.Src, src.Path().ConnInit(dst.Path(), addr))
		// Handshake has started on dst (1 stepdone), relay `connOpenTry` and `updateClient` on src
	case srcConn.Connection.State == conntypes.UNINITIALIZED && dstConn.Connection.State == conntypes.INIT:
		logConnectionStates(src, dst, srcConn, dstConn)
		addr := mustGetAddress(src)
		if dstUpdateHeader != nil {
			out.Src = append(out.Src, src.Path().UpdateClient(dstUpdateHeader, addr))
		}
		out.Src = append(out.Src, src.Path().ConnTry(dst.Path(), dstCsRes, dstConn, dstCons, addr))
	// Handshake has started on src (1 step done), relay `connOpenTry` and `updateClient` on dst
	case srcConn.Connection.State == conntypes.INIT && dstConn.Connection.State == conntypes.UNINITIALIZED:
		logConnectionStates(dst, src, dstConn, srcConn)
		addr := mustGetAddress(dst)
		if srcUpdateHeader != nil {
			out.Dst = append(out.Dst, dst.Path().UpdateClient(srcUpdateHeader, addr))
		}
		out.Dst = append(out.Dst, dst.Path().ConnTry(src.Path(), srcCsRes, srcConn, srcCons, addr))

	// Handshake has started on src end (2 steps done), relay `connOpenAck` and `updateClient` to dst end
	case srcConn.Connection.State == conntypes.TRYOPEN && dstConn.Connection.State == conntypes.INIT:
		logConnectionStates(dst, src, dstConn, srcConn)
		addr := mustGetAddress(dst)
		if srcUpdateHeader != nil {
			out.Dst = append(out.Dst, dst.Path().UpdateClient(srcUpdateHeader, addr))
		}
		out.Dst = append(out.Dst, dst.Path().ConnAck(src.Path(), srcCsRes, srcConn, srcCons, addr))

	// Handshake has started on dst end (2 steps done), relay `connOpenAck` and `updateClient` to src end
	case srcConn.Connection.State == conntypes.INIT && dstConn.Connection.State == conntypes.TRYOPEN:
		logConnectionStates(src, dst, srcConn, dstConn)
		addr := mustGetAddress(src)
		if dstUpdateHeader != nil {
			out.Src = append(out.Src, src.Path().UpdateClient(dstUpdateHeader, addr))
		}
		out.Src = append(out.Src, src.Path().ConnAck(dst.Path(), dstCsRes, dstConn, dstCons, addr))

	// Handshake has confirmed on dst (3 steps done), relay `connOpenConfirm` and `updateClient` to src end
	case srcConn.Connection.State == conntypes.TRYOPEN && dstConn.Connection.State == conntypes.OPEN:
		logConnectionStates(src, dst, srcConn, dstConn)
		addr := mustGetAddress(src)
		if dstUpdateHeader != nil {
			out.Src = append(out.Src, src.Path().UpdateClient(dstUpdateHeader, addr))
		}
		out.Src = append(out.Src, src.Path().ConnConfirm(dstConn, addr))
		out.Last = true

	// Handshake has confirmed on src (3 steps done), relay `connOpenConfirm` and `updateClient` to dst end
	case srcConn.Connection.State == conntypes.OPEN && dstConn.Connection.State == conntypes.TRYOPEN:
		logConnectionStates(dst, src, dstConn, srcConn)
		addr := mustGetAddress(dst)
		if srcUpdateHeader != nil {
			out.Dst = append(out.Dst, dst.Path().UpdateClient(srcUpdateHeader, addr))
		}
		out.Dst = append(out.Dst, dst.Path().ConnConfirm(srcConn, addr))
		out.Last = true

	default:
		panic(fmt.Sprintf("not implemented error: %v %v", srcConn.Connection.State, dstConn.Connection.State))
	}

	return out, nil
}

// validatePaths takes two chains and validates their paths
func validatePaths(src, dst ChainI) error {
	if err := src.Path().Validate(); err != nil {
		return fmt.Errorf("path on chain %s failed to set: %w", src.ChainID(), err)
	}
	if err := dst.Path().Validate(); err != nil {
		return fmt.Errorf("path on chain %s failed to set: %w", dst.ChainID(), err)
	}
	return nil
}

func logConnectionStates(src, dst ChainI, srcConn, dstConn *conntypes.QueryConnectionResponse) {
	log.Println(fmt.Sprintf("- [%s]@{%d}conn(%s)-{%s} : [%s]@{%d}conn(%s)-{%s}",
		src.ChainID(),
		mustGetHeight(srcConn.ProofHeight),
		src.Path().ConnectionID,
		srcConn.Connection.State,
		dst.ChainID(),
		mustGetHeight(dstConn.ProofHeight),
		dst.Path().ConnectionID,
		dstConn.Connection.State,
	))
}

// mustGetHeight takes the height inteface and returns the actual height
func mustGetHeight(h ibcexported.Height) uint64 {
	height, ok := h.(clienttypes.Height)
	if !ok {
		panic("height is not an instance of height! wtf")
	}
	return height.GetRevisionHeight()
}

func mustGetAddress(chain interface {
	GetAddress() (sdk.AccAddress, error)
}) sdk.AccAddress {
	addr, err := chain.GetAddress()
	if err != nil {
		panic(err)
	}
	return addr
}
