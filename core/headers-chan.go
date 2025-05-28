package core

func DrainHeadersChan(headers <-chan Header) []Header {
	var ret []Header
	for h := range headers {
		ret = append(ret, h)
	}
	return ret
}

func MakeHeadersChan(headers ...Header) <-chan Header {
	ch := make(chan Header, len(headers))
	for _, h := range headers {
		ch <- h
	}
	close(ch)
	return ch
}
