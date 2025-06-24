package core

import "context"

import (
	"os"
	"strconv"
	"fmt"
	"time"
)

func SetupHeadersForUpdateSync(prover LightClient, ctx context.Context, counterparty FinalityAwareChain, latestFinalizedHeader Header) ([]Header, error) {
	ctxForSHFU, cancel := context.WithCancel(ctx)
	defer cancel()
	headerStream, err := prover.SetupHeadersForUpdate(ctxForSHFU, counterparty, latestFinalizedHeader)
	if err != nil {
		return nil, err
	}

	var ret []Header
	for h := range headerStream {
		if h.Error != nil {
			return nil, h.Error
		}
		ret = append(ret, h.Header)
	}
	return ret, nil
}

/*
func MakeHeaderStream(headers ...Header) <-chan *HeaderOrError {
	ch := make(chan *HeaderOrError, len(headers))
	for _, h := range headers {
		ch <- &HeaderOrError{
			Header: h,
			Error:  nil,
		}
	}
	close(ch)
	return ch
}
*/
func MakeHeaderStream(headers ...Header) <-chan *HeaderOrError {
	ch := make(chan *HeaderOrError, len(headers))
	go func() {
		for _, h := range headers {
			ch <- &HeaderOrError{
				Header: h,
				Error:  nil,
			}
		}
		if val, ok := os.LookupEnv("DEBUG_RELAYER_WAIT"); ok {
			i, _ := strconv.Atoi(val)
			fmt.Printf(">DEBUG_RELAYER_WAIT: %v\n", i)
			time.Sleep(time.Duration(i) * time.Second)
			fmt.Printf("<DEBUG_RELAYER_WAIT: %v\n", i)
		} else {
			fmt.Printf("DEBUG_RELAYER_WAIT is not set\n")
		}
		close(ch)
	}()
	return ch
}
