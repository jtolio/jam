package blobs

import (
	"bufio"
	"context"
	"io"
	"sort"

	"github.com/zeebo/errs"

	"github.com/jtolio/jam/backends"
	"github.com/jtolio/jam/manifest"
	"github.com/jtolio/jam/streams"
)

type entry struct {
	source  io.ReadCloser
	cb      func(ctx context.Context, stream *manifest.Stream, lastOfBlob bool) error
	size    int64
	sortKey SortKey
	stream  *manifest.Stream
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

type SortKey interface {
	Less(b SortKey) bool
}

func (s *Store) Put(ctx context.Context, data io.ReadCloser, size int64, sortKey SortKey, cb func(ctx context.Context, stream *manifest.Stream, lastOfBlob bool) error) error {
	s.unflushed = append(s.unflushed, &entry{
		source:  data,
		cb:      cb,
		size:    size,
		sortKey: sortKey,
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

	sort.Sort(entriesBySortKey(unflushed))

	c := newConcat(unflushed...)

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
		err = c.Cut(ctx)
		if err != nil {
			return errs.Wrap(err)
		}
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

type entriesBySortKey []*entry

func (e entriesBySortKey) Len() int           { return len(e) }
func (e entriesBySortKey) Swap(i, j int)      { e[i], e[j] = e[j], e[i] }
func (e entriesBySortKey) Less(i, j int) bool { return e[i].sortKey.Less(e[j].sortKey) }
