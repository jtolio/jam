package pathdb

import (
	"compress/gzip"
	"context"
	"io"

	"github.com/jtolds/jam/backends"
	"github.com/jtolds/jam/pkg/manifest"
	"github.com/jtolds/jam/pkg/streams"
	"github.com/zeebo/errs"
)

type DB struct {
	backend backends.Backend
}

func Open(ctx context.Context, backend backends.Backend, stream io.Reader) (*DB, error) {
	db := &DB{
		backend: backend,
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
				panic("TODO")
				_ = entry
			}
		}
	}

	return nil
}
