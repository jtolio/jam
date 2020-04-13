package hashdb

import (
	"context"

	"github.com/jtolds/jam/manifest"
)

type DB struct{}

func New() *DB { return &DB{} }

func (d *DB) Has(ctx context.Context, hash string) (exists bool, err error) {
	panic("TODO")
}

func (d *DB) Put(ctx context.Context, hash string, data *manifest.Stream) error {
	panic("TODO")
}

func (d *DB) Flush(ctx context.Context) error {
	panic("TODO")
}
