package pathdb

import (
	"context"

	"github.com/jtolds/jam/backends"
)

type DB struct{}

func Open(ctx context.Context, backend backends.Backend, root string) (*DB, error) {
	panic("TODO")
}
