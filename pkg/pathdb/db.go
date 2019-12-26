package pathdb

import (
	"compress/gzip"
	"context"
	"io"
	"strings"

	"github.com/zeebo/errs"
	"modernc.org/b"

	"github.com/jtolds/jam/backends"
	"github.com/jtolds/jam/pkg/blobs"
	"github.com/jtolds/jam/pkg/manifest"
	"github.com/jtolds/jam/pkg/streams"
)

type DB struct {
	backend backends.Backend
	blobs   *blobs.Store
	tree    *b.Tree
}

func Open(ctx context.Context, backend backends.Backend, blobStore *blobs.Store, stream io.Reader) (*DB, error) {
	db := &DB{
		backend: backend,
		blobs:   blobStore,
		tree: b.TreeNew(func(a, b interface{}) int {
			return strings.Compare(a.(string), b.(string))
		}),
	}
	return db, db.load(ctx, stream)
}

func (db *DB) load(ctx context.Context, stream io.Reader) error {
	r, err := gzip.NewReader(stream)
	if err != nil {
		return err
	}
	defer func() {
		err = errs.Combine(err, r.Close())
	}()

	for {
		var page manifest.Page
		err := manifest.UnmarshalSized(r, &page)
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		if branch := page.GetBranch(); branch != nil {
			substream, err := streams.Open(ctx, db.backend, branch)
			if err != nil {
				return err
			}
			err = db.load(ctx, substream)
			if err != nil {
				return err
			}
		}
		if entries := page.GetEntries(); entries != nil {
			for _, entry := range entries.Entries {
				db.tree.Set(entry.Path, entry.Content)
			}
		}
	}

	return nil
}

func (db *DB) Get(ctx context.Context, path string) (*manifest.Content, error) {
	v, ok := db.tree.Get(path)
	if !ok {
		return nil, nil
	}
	return v.(*manifest.Content), nil
}

func (db *DB) List(ctx context.Context, prefix string, recursive bool,
	cb func(ctx context.Context, path string, content *manifest.Content) error) error {
	panic("TODO")
}

func (db *DB) Put(ctx context.Context, path string, content *manifest.Content) error {
	db.tree.Set(path, content)
	return nil
}

func (db *DB) Delete(ctx context.Context, path string) error {
	db.tree.Delete(path)
	return nil
}

func (db *DB) Save(ctx context.Context, backendPath string) error {
	// TODO: weird that this is the only place that backend paths leak into
	// this data structure. should we harmonize how open and save work?
	// Perhaps Saving should be an external method that operates on a pathdb?
	panic("TODO")
}
