package streams

import (
	"context"
	"fmt"
	"io"

	"github.com/jtolds/jam/backends"
	"github.com/jtolds/jam/pkg/manifest"
)

const BlobPrefix = "blob/"

func openRange(ctx context.Context, backend backends.Backend, r *manifest.Range, offset int64) (io.ReadCloser, error) {
	if offset > r.Length {
		return nil, fmt.Errorf("invalid offset")
	}
	rc, err := backend.Get(ctx, BlobPrefix+r.Blob, r.Offset+offset)
	if err != nil {
		return nil, err
	}
	return struct {
		io.Reader
		io.Closer
	}{
		Reader: io.LimitReader(rc, r.Length-offset),
		Closer: rc,
	}, nil
}
