package core

import "golang.org/x/sync/errgroup"

// UpdatesWithHeaders calls UpdateLightWithHeader on the passed chains concurrently
func UpdatesWithHeaders(src, dst LightClientI) (srch, dsth HeaderI, err error) {
	var eg = new(errgroup.Group)
	eg.Go(func() error {
		srch, err = src.QueryLatestHeader()
		return err
	})
	eg.Go(func() error {
		dsth, err = dst.QueryLatestHeader()
		return err
	})
	if err := eg.Wait(); err != nil {
		return nil, nil, err
	}
	return srch, dsth, nil
}
