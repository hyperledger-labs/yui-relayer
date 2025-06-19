package core

import "context"

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
