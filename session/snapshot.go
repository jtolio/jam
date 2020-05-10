package session

import (
	"context"
	"fmt"

	"github.com/jtolds/jam/backends"
	"github.com/jtolds/jam/blobs"
	"github.com/jtolds/jam/hashdb"
	"github.com/jtolds/jam/manifest"
	"github.com/jtolds/jam/pathdb"
	"github.com/jtolds/jam/streams"
	"github.com/zeebo/errs"
)

type Snapshot struct {
	backend backends.Backend
	paths   *pathdb.DB
	blobs   *blobs.Store
	hashes  *hashdb.DB
}

func newSnapshot(backend backends.Backend, paths *pathdb.DB, blobStore *blobs.Store, hashes *hashdb.DB) *Snapshot {
	return &Snapshot{
		backend: backend,
		paths:   paths,
		blobs:   blobStore,
		hashes:  hashes,
	}
}

type ListEntry struct {
	Path   string
	Prefix bool
	Meta   *manifest.Metadata

	backend backends.Backend
	data    *manifest.Stream
}

func (e *ListEntry) Stream(ctx context.Context) (*streams.Stream, error) {
	return streams.Open(ctx, e.backend, e.data)
}

func (s *Snapshot) List(ctx context.Context, prefix, delimiter string,
	cb func(ctx context.Context, entry *ListEntry) error) error {
	return s.paths.List(ctx, prefix, delimiter,
		func(ctx context.Context, path string, content *manifest.Content) error {
			if content == nil {
				return cb(ctx, &ListEntry{Path: path, Prefix: true})
			}

			data, err := s.getStream(ctx, content)
			if err != nil {
				return err
			}

			return cb(ctx, &ListEntry{Path: path, Meta: content.Metadata, backend: s.backend, data: data})
		})
}

var ErrNotFound = fmt.Errorf("file not found")

func (s *Snapshot) getStream(ctx context.Context, content *manifest.Content) (*manifest.Stream, error) {
	if content.Metadata.Type != manifest.Metadata_FILE {
		return nil, nil
	}

	if content.Data != nil {
		return content.Data, nil
	}
	if len(content.Hash) == 0 {
		return nil, errs.New("no content found, but content expected")
	}
	data, err := s.hashes.Lookup(ctx, string(content.Hash))
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, errs.New("hash not found")
	}
	return data, nil
}

// Open will return a nil stream if the filetype is a symlink or something
func (s *Snapshot) Open(ctx context.Context, path string) (*manifest.Metadata, *streams.Stream, error) {
	content, err := s.paths.Get(ctx, path)
	if err != nil {
		return nil, nil, err
	}
	if content == nil {
		return nil, nil, ErrNotFound
	}

	data, err := s.getStream(ctx, content)
	if err != nil {
		return nil, nil, err
	}
	if data != nil {
		stream, err := streams.Open(ctx, s.backend, data)
		return content.Metadata, stream, err
	}
	return content.Metadata, nil, nil
}

func (s *Snapshot) HasPrefix(ctx context.Context, prefix string) (exists bool, err error) {
	return s.paths.HasPrefix(ctx, prefix)
}

func (s *Snapshot) Close() error {
	return s.paths.Close()
}
