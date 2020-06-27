package backends

import (
	"context"
	"io"

	"github.com/jtolds/jam/utils"
)

type combined struct {
	backends    []Backend
	compareGets bool
}

// Combine takes a set of 1 or more backends and combines them such that
// Gets and Lists go to just the primary backend, and Puts, Deletes, and
// Closes go to all backends
func Combine(primary Backend, others ...Backend) Backend {
	return newCombined(primary, others, false)
}

// CombineAndCompare is like Combine, but Gets will read from all backends
// simultaneously, returning an error if any of the resulting data does not
// match.
func CombineAndCompare(primary Backend, others ...Backend) Backend {
	return newCombined(primary, others, true)
}

func newCombined(primary Backend, others []Backend, compareGets bool) *combined {
	return &combined{
		backends: append(append(
			make([]Backend, 0, len(others)+1),
			primary),
			others...),
		compareGets: compareGets,
	}
}

var _ Backend = (*combined)(nil)

func (c *combined) Get(ctx context.Context, path string, offset, length int64) (io.ReadCloser, error) {
	if !c.compareGets {
		return c.backends[0].Get(ctx, path, offset, length)
	}

	readers := make([]io.ReadCloser, 0, len(c.backends))
	for _, b := range c.backends {
		rc, err := b.Get(ctx, path, offset, length)
		if err != nil {
			for _, or := range readers {
				or.Close()
			}
			return nil, err
		}
		readers = append(readers, rc)
	}

	return utils.ReaderCompare(readers...), nil
}

func (c *combined) List(ctx context.Context, prefix string, cb func(ctx context.Context, path string) error) error {
	return c.backends[0].List(ctx, prefix, cb)
}

func (c *combined) Put(ctx context.Context, path string, data io.Reader) error {
	fns := make([]func() error, 0, len(c.backends))
	current := c.backends[0]
	remaining := c.backends[1:]
	for len(remaining) > 0 {
		r1, r2 := utils.ReaderTee(data)
		data = r1
		func(o Backend, r io.Reader) { // range variable/closure fix
			fns = append(fns, func() error {
				return o.Put(ctx, path, r)
			})
		}(current, r2)
		current = remaining[0]
		remaining = remaining[1:]
	}
	fns = append(fns, func() error {
		return current.Put(ctx, path, data)
	})
	err := utils.Parallel(fns...)
	if err != nil {
		c.Delete(ctx, path)
		return err
	}
	return nil
}

func (c *combined) Delete(ctx context.Context, path string) error {
	fns := make([]func() error, 0, len(c.backends))
	for _, o := range c.backends {
		func(o Backend) { // range variable/closure fix
			fns = append(fns, func() error {
				return o.Delete(ctx, path)
			})
		}(o)
	}
	return utils.Parallel(fns...)

}

func (c *combined) Close() error {
	fns := make([]func() error, 0, len(c.backends))
	for _, o := range c.backends {
		fns = append(fns, o.Close)
	}
	return utils.Parallel(fns...)
}
