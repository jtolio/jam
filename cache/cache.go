package cache

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/jtolds/jam/backends"
	"github.com/zeebo/errs"
)

// Cache is a write-through blob cache using the Misra Gries heavy hitter summary
// to determine which blobs to cache.
type Cache struct {
	persistent  backends.Backend
	cache       backends.Backend
	readMax     int64
	cacheSize   int
	mtx         sync.Mutex
	cached      map[string]bool
	misraGries  map[string]int64
	openHandles map[string]int64
}

func New(ctx context.Context, persistent, cache backends.Backend, readMax int64, cacheSize int) (*Cache, error) {
	if readMax <= 0 || cacheSize <= 0 {
		return nil, fmt.Errorf("invalid configuration")
	}
	c := &Cache{
		persistent:  persistent,
		cache:       cache,
		readMax:     readMax,
		cacheSize:   cacheSize,
		cached:      map[string]bool{},
		misraGries:  map[string]int64{},
		openHandles: map[string]int64{},
	}
	return c, c.cache.List(ctx, "", func(ctx context.Context, path string) error {
		c.cached[path] = true
		return nil
	})
}

var _ backends.Backend = (*Cache)(nil)

func (c *Cache) Get(ctx context.Context, path string, offset, length int64) (io.ReadCloser, error) {
	c.mtx.Lock()

	evict := func(path string) error {
		delete(c.cached, path)
		delete(c.openHandles, path)
		return c.cache.Delete(ctx, path)
	}

	serveFromCache := func() (io.ReadCloser, error) {
		rc, err := c.cache.Get(ctx, path, offset, length)
		if err != nil {
			c.mtx.Unlock()
			return nil, err
		}
		c.openHandles[path]++
		c.mtx.Unlock()
		return struct {
			io.Reader
			io.Closer
		}{
			Reader: rc,
			Closer: closerFunc(func() error {
				closeErr := rc.Close()
				var uncacheErr error
				c.mtx.Lock()
				c.openHandles[path]--
				if c.openHandles[path] <= 0 && c.misraGries[path] <= 0 {
					uncacheErr = evict(path)
				}
				c.mtx.Unlock()
				return errs.Combine(closeErr, uncacheErr)
			}),
		}, nil
	}

	fillCache := func() error {
		if c.cached[path] {
			return nil
		}
		for overflow := len(c.cached) - c.cacheSize; overflow > 0; overflow-- {
			for cachedPath := range c.cached {
				if c.misraGries[cachedPath] > 0 ||
					c.openHandles[cachedPath] > 0 {
					continue
				}
				err := evict(cachedPath)
				if err != nil {
					return err
				}
				break
			}
		}

		rc, err := c.persistent.Get(ctx, path, 0, -1)
		if err != nil {
			return err
		}
		defer rc.Close()
		err = c.cache.Put(ctx, path, rc)
		if err != nil {
			return err
		}
		c.cached[path] = true
		return nil
	}

	if length < 0 || length > c.readMax {
		if c.cached[path] {
			return serveFromCache()
		}
		c.mtx.Unlock()
		return c.persistent.Get(ctx, path, offset, length)
	}

	if count, exists := c.misraGries[path]; exists {
		// the path is already a heavy hitter. load from cache.
		c.misraGries[path] = count + 1
		return serveFromCache()

	}

	if len(c.misraGries) < c.cacheSize-1 {
		// the path just became a heavy hitter.
		err := fillCache()
		if err != nil {
			c.mtx.Unlock()
			return nil, err
		}
		c.misraGries[path] = 1
		return serveFromCache()
	}

	// the path is not a heavy hitter. update misra gries
	for path, count := range c.misraGries {
		count--
		if count > 0 {
			c.misraGries[path] = count
		} else {
			delete(c.misraGries, path)
		}
	}

	if c.cached[path] {
		// it's still in the cache anyway
		return serveFromCache()
	}

	c.mtx.Unlock()

	// just serve from persistent storage i guess
	return c.persistent.Get(ctx, path, offset, length)
}

type closerFunc func() error

func (f closerFunc) Close() error { return f() }

func (c *Cache) Put(ctx context.Context, path string, data io.Reader) error {
	return c.persistent.Put(ctx, path, data)
}

func (c *Cache) Delete(ctx context.Context, path string) error {
	c.mtx.Lock()
	delete(c.misraGries, path)
	delete(c.cached, path)
	delete(c.openHandles, path)
	err := c.cache.Delete(ctx, path)
	c.mtx.Unlock()
	return errs.Combine(c.persistent.Delete(ctx, path), err)
}

func (c *Cache) List(ctx context.Context, prefix string, cb func(ctx context.Context, path string) error) error {
	return c.persistent.List(ctx, prefix, cb)
}

func (c *Cache) Close() error {
	return errs.Combine(c.persistent.Close(), c.cache.Close())
}
