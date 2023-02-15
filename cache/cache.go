package cache

import (
	"context"
	"errors"
	"io"
	"strings"

	"github.com/jtolio/jam/backends"
	"github.com/jtolio/jam/hashdb"
	"github.com/jtolio/jam/session"
)

type Cache struct {
	persistent backends.Backend
	cache      backends.Backend
	both       backends.Backend
	cacheBlobs bool
}

func New(ctx context.Context, persistent, cache backends.Backend, cacheBlobs bool) (*Cache, error) {
	c := &Cache{
		persistent: persistent,
		cache:      cache,
		both:       backends.Combine(persistent, cache),
		cacheBlobs: cacheBlobs,
	}

	return c, nil
}

var _ backends.Backend = (*Cache)(nil)

func (c *Cache) Get(ctx context.Context, path string, offset, length int64) (io.ReadCloser, error) {
	rc, err := c.cache.Get(ctx, path, offset, length)
	if err == nil {
		return rc, nil
	}
	if !errors.Is(err, backends.ErrNotExist) {
		return nil, err
	}

	if c.shouldCache(path) {
		rc, err := c.persistent.Get(ctx, path, 0, -1)
		if err != nil {
			return nil, err
		}
		defer rc.Close()
		err = c.cache.Put(ctx, path, rc)
		if err != nil {
			return nil, err
		}
		return c.cache.Get(ctx, path, offset, length)
	}

	return c.persistent.Get(ctx, path, offset, length)
}

func (c *Cache) Put(ctx context.Context, path string, data io.Reader) error {
	if c.shouldCache(path) {
		return c.both.Put(ctx, path, data)
	}
	return c.persistent.Put(ctx, path, data)
}

func (c *Cache) Delete(ctx context.Context, path string) error {
	return c.both.Delete(ctx, path)
}

func (c *Cache) List(ctx context.Context, prefix string, cb func(ctx context.Context, path string) error) error {
	return c.persistent.List(ctx, prefix, cb)
}

func (c *Cache) Close() error {
	return c.both.Close()
}

func (c *Cache) shouldCache(path string) bool {
	if strings.HasPrefix(path, hashdb.HashPrefix) {
		return true
	}
	if strings.HasPrefix(path, session.ManifestPrefix) {
		return true
	}
	return c.cacheBlobs
}
