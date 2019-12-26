package blobs

import (
	"bufio"
	"context"
	"io"

	"github.com/zeebo/errs"

	"github.com/jtolds/jam/backends"
	"github.com/jtolds/jam/pkg/manifest"
)

type entry struct {
	source io.ReadCloser
	cb     func(*manifest.Stream)
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

func (s *Store) Put(ctx context.Context, data io.ReadCloser, cb func(*manifest.Stream)) error {
	s.unflushed = append(s.unflushed, &entry{
		source: data,
		cb:     cb,
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

	c := newConcat(unflushed...)

	for !c.EOF() {
		blob := bufio.NewReader(io.LimitReader(c, s.blobSize))
		_, err = blob.Peek(1)
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		err = s.backend.Put(ctx, blobPath(c.Blob()), blob)
		if err != nil {
			return err
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
		group.Add(entry.source.Close())
	}
	return group.Err()
}
