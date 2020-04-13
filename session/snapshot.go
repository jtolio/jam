package session

import (
	"context"
	"fmt"

	"github.com/jtolds/jam/backends"
	"github.com/jtolds/jam/blobs"
	"github.com/jtolds/jam/manifest"
	"github.com/jtolds/jam/pathdb"
	"github.com/jtolds/jam/streams"
)

type Snapshot struct {
	backend backends.Backend
	paths   *pathdb.DB
	blobs   *blobs.Store
}

func newSnapshot(backend backends.Backend, paths *pathdb.DB, blobStore *blobs.Store) *Snapshot {
	return &Snapshot{
		backend: backend,
		paths:   paths,
		blobs:   blobStore,
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
			return cb(ctx, &ListEntry{Path: path, Meta: content.Metadata, backend: s.backend, data: content.Data})
		})
}

var ErrNotFound = fmt.Errorf("file not found")

// Open will return a nil stream if the filetype is a symlink or something
func (s *Snapshot) Open(ctx context.Context, path string) (*manifest.Metadata, *streams.Stream, error) {
	content, err := s.paths.Get(ctx, path)
	if err != nil {
		return nil, nil, err
	}
	if content == nil {
		return nil, nil, ErrNotFound
	}
	if content.Metadata.Type == manifest.Metadata_FILE {
		stream, err := streams.Open(ctx, s.backend, content.Data)
		return content.Metadata, stream, err
	}
	return content.Metadata, nil, err
}

func (s *Snapshot) HasPrefix(ctx context.Context, prefix string) (exists bool, err error) {
	return s.paths.HasPrefix(ctx, prefix)
}

func (s *Snapshot) Close() error {
	return s.paths.Close()
}
