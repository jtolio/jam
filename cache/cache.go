package cache

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/zeebo/errs"

	"github.com/jtolds/jam/backends"
	"github.com/jtolds/jam/utils"
)

const versionHeader = "jam-v0\n"

// Cache is a write-through LRU blob cache using a capped Misra Gries heavy
// hitter summary to determine which blobs to add to the LRU.
type Cache struct {
	persistent     backends.Backend
	cache          backends.Backend
	cacheStateFile string

	mtx         sync.Mutex
	mg          *cappedMisraGries
	lru         *lru
	openHandles map[string]int
	cached      map[string]bool
}

func New(ctx context.Context, persistent, cache backends.Backend, cacheSize, minHits int, cacheStateFile string) (*Cache, error) {
	mg, err := newCappedMisraGries(cacheSize, minHits)
	if err != nil {
		return nil, err
	}
	c := &Cache{
		persistent:     persistent,
		cache:          cache,
		cacheStateFile: cacheStateFile,
		mg:             mg,
		lru:            newLRU(cacheSize),
		openHandles:    map[string]int{},
		cached:         map[string]bool{},
	}

	err = c.load(ctx)
	if err != nil {
		return nil, err
	}

	return c, nil
}

var _ backends.Backend = (*Cache)(nil)

func (c *Cache) load(ctx context.Context) error {
	// make sure we know about existing cached objects
	err := c.cache.List(ctx, "", func(ctx context.Context, path string) error {
		c.cached[path] = true
		if evicted, eviction := c.lru.Put(path); eviction {
			delete(c.cached, evicted)
			err := c.cache.Delete(ctx, evicted)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	err = c.loadCache(ctx)
	if err != nil {
		utils.L(ctx).Normalf("cache is bad, ignoring: %v", err)
	}
	return nil
}

func (c *Cache) loadCache(ctx context.Context) error {
	if c.cacheStateFile == "" {
		return nil
	}
	fh, err := os.Open(c.cacheStateFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer fh.Close()

	// TODO: reduce code duplication with pathdb.load
	v := make([]byte, len([]byte(versionHeader)))
	_, err = io.ReadFull(fh, v)
	if err != nil {
		return err
	}
	if versionHeader != string(v) {
		return fmt.Errorf("bad version")
	}

	var pb CacheState
	err = utils.UnmarshalSized(fh, &pb)
	if err != nil {
		return err
	}

	c.mg.Load(pb.Counts)
	c.lru.Load(pb.Lru)

	return nil
}

func (c *Cache) Get(ctx context.Context, path string, offset, length int64) (io.ReadCloser, error) {
	c.mtx.Lock()

	if c.mg.Observe(path) {
		if evicted, eviction := c.lru.Put(path); eviction {
			err := c.checkEvict(ctx, evicted)
			if err != nil {
				c.mtx.Unlock()
				return nil, err
			}
		}

		if !c.cached[path] {
			// TODO: we maybe can avoid doing this under a lock
			rc, err := c.persistent.Get(ctx, path, 0, -1)
			if err != nil {
				c.mtx.Unlock()
				return nil, err
			}
			err = errs.Combine(
				c.cache.Put(ctx, path, rc),
				rc.Close())
			if err != nil {
				c.mtx.Unlock()
				return nil, err
			}
			c.cached[path] = true
		}

	} else if !c.cached[path] {
		c.mtx.Unlock()
		return c.persistent.Get(ctx, path, offset, length)
	}

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
			c.mtx.Lock()
			count := c.openHandles[path] - 1
			if count > 0 {
				c.openHandles[path] = count
			} else {
				delete(c.openHandles, path)
			}
			err := c.checkEvict(ctx, path)
			c.mtx.Unlock()
			return errs.Combine(rc.Close(), err)
		}),
	}, nil
}

type closerFunc func() error

func (f closerFunc) Close() error { return f() }

func (c *Cache) checkEvict(ctx context.Context, path string) error {
	if c.openHandles[path] > 0 {
		return nil
	}
	if c.lru.Has(path) {
		return nil
	}
	if !c.cached[path] {
		return nil
	}
	delete(c.cached, path)
	return c.cache.Delete(ctx, path)
}

func (c *Cache) Put(ctx context.Context, path string, data io.Reader) error {
	return c.persistent.Put(ctx, path, data)
}

func (c *Cache) Delete(ctx context.Context, path string) error {
	c.mtx.Lock()
	c.mg.Delete(path)
	c.lru.Remove(path)
	var deleteErr error
	if c.cached[path] {
		delete(c.cached, path)
		deleteErr = c.cache.Delete(ctx, path)
	}
	c.mtx.Unlock()
	return errs.Combine(c.persistent.Delete(ctx, path), deleteErr)
}

func (c *Cache) List(ctx context.Context, prefix string, cb func(ctx context.Context, path string) error) error {
	return c.persistent.List(ctx, prefix, cb)
}

func (c *Cache) Close() error {
	return errs.Combine(c.saveCache(), c.persistent.Close(), c.cache.Close())
}

func (c *Cache) saveCache() error {
	if c.cacheStateFile == "" {
		return nil
	}
	cs := CacheState{
		Counts: c.mg.Save(),
		Lru:    c.lru.Save(),
	}

	serialized, err := utils.MarshalSized(&cs)
	if err != nil {
		return err
	}

	fh, err := os.Create(c.cacheStateFile)
	if err != nil {
		return err
	}
	defer fh.Close()

	_, err = io.Copy(fh, io.MultiReader(
		strings.NewReader(versionHeader),
		bytes.NewReader(serialized)))
	if err != nil {
		return err
	}

	return fh.Close()
}
