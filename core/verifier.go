package core

import (
	"context"

	"golang.org/x/sync/errgroup"
)

// UpdatesWithHeaders calls UpdateLightWithHeader on the passed chains concurrently
func UpdatesWithHeaders(ctx context.Context, src, dst ChainI) (srch, dsth HeaderI, err error) {
	var eg = new(errgroup.Group)
	eg.Go(func() error {
		srch, err = src.QueryLatestHeader(ctx)
		return err
	})
	eg.Go(func() error {
		dsth, err = dst.QueryLatestHeader(ctx)
		return err
	})
	if err := eg.Wait(); err != nil {
		return nil, nil, err
	}
	return srch, dsth, nil
}
