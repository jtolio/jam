package backends

import (
	"context"
	"io"

	"github.com/jtolds/jam/utils"
	"github.com/zeebo/errs"
)

type combined struct {
	primary Backend
	others  []Backend
}

// Combine takes a set of 1 or more backends and combines them such that
// Gets and Lists go to just the primary backend, and Puts, Deletes, and
// Closes go to all backends
func Combine(primary Backend, others ...Backend) Backend {
	return &combined{primary: primary, others: others}
}

var _ Backend = (*combined)(nil)

func (c *combined) Get(ctx context.Context, path string, offset int64) (io.ReadCloser, error) {
	return c.primary.Get(ctx, path, offset)
}

func (c *combined) List(ctx context.Context, prefix string, cb func(ctx context.Context, path string) error) error {
	return c.primary.List(ctx, prefix, cb)
}

func (c *combined) Put(ctx context.Context, path string, data io.Reader) error {
	fns := make([]func() error, 0, 1+len(c.others))
	current := c.primary
	remaining := append([]Backend(nil), c.others...)
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
	return parallel(fns...)
}

func (c *combined) Delete(ctx context.Context, path string) error {
	fns := make([]func() error, 0, 1+len(c.others))
	fns = append(fns, func() error {
		return c.primary.Delete(ctx, path)
	})
	for _, o := range c.others {
		func(o Backend) { // range variable/closure fix
			fns = append(fns, func() error {
				return o.Delete(ctx, path)
			})
		}(o)
	}
	return parallel(fns...)

}

func (c *combined) Close() error {
	fns := make([]func() error, 0, 1+len(c.others))
	fns = append(fns, c.primary.Close)
	for _, o := range c.others {
		fns = append(fns, o.Close)
	}
	return parallel(fns...)
}

func parallel(fn ...func() error) error {
	errch := make(chan error, len(fn))
	for _, f := range fn {
		go func(f func() error) {
			errch <- f()
		}(f)
	}
	var eg errs.Group
	for range fn {
		eg.Add(<-errch)
	}
	return eg.Err()
}
