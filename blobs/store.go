package blobs

import (
	"bufio"
	"context"
	"io"
	"sort"

	"github.com/zeebo/errs"

	"github.com/jtolds/jam/backends"
	"github.com/jtolds/jam/manifest"
	"github.com/jtolds/jam/streams"
)

type entry struct {
	source io.ReadCloser
	cb     func(context.Context, *manifest.Stream) error
	size   int64
}

type Store struct {
	backend      backends.Backend
	blobSize     int64
	maxUnflushed int
	unflushed    []*entry
}

func NewStore(backend backends.Backend, blobSize int64, maxUnflushed int) *Store {
	return &Store{
		backend:      backend,
		blobSize:     blobSize,
		maxUnflushed: maxUnflushed,
	}
}

func (s *Store) Put(ctx context.Context, data io.ReadCloser, size int64, cb func(context.Context, *manifest.Stream) error) error {
	s.unflushed = append(s.unflushed, &entry{
		source: data,
		cb:     cb,
		size:   size,
	})
	if len(s.unflushed) <= s.maxUnflushed {
		return nil
	}
	return s.Flush(ctx)
}

func (s *Store) Flush(ctx context.Context) (err error) {
	unflushed := s.unflushed
	s.unflushed = nil
	defer func() {
		err = errs.Combine(err, closeEntries(unflushed))
	}()

	sort.Sort(entriesBySize(unflushed))

	c, err := newConcat(ctx, unflushed...)
	if err != nil {
		return err
	}

	for !c.EOF() {
		blob := bufio.NewReader(io.LimitReader(c, s.blobSize))
		_, err = blob.Peek(1)
		if err != nil {
			if err == io.EOF {
				break
			}
			return errs.Wrap(err)
		}
		err = s.backend.Put(ctx, streams.BlobPath(c.Blob()), blob)
		if err != nil {
			return errs.Wrap(err)
		}
		c.Cut()
	}

	return nil
}

func (s *Store) Close() error {
	unflushed := s.unflushed
	s.unflushed = nil
	return closeEntries(unflushed)
}

func closeEntries(entries []*entry) error {
	var group errs.Group
	for _, entry := range entries {
		group.Add(errs.Wrap(entry.source.Close()))
	}
	return group.Err()
}

type entriesBySize []*entry

func (e entriesBySize) Len() int           { return len(e) }
func (e entriesBySize) Swap(i, j int)      { e[i], e[j] = e[j], e[i] }
func (e entriesBySize) Less(i, j int) bool { return e[i].size < e[j].size }
