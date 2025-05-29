package core

func DrainHeaderStream(headers <-chan *HeaderOrError) ([]Header, error) {
	var ret []Header
	for h := range headers {
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
