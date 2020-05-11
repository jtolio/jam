package streams

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/jtolds/jam/backends"
	"github.com/jtolds/jam/manifest"
)

const BlobPrefix = "blob/"

func BlobPath(id string) string {
	return BlobPrefix + id[:2] + "/" + id[2:]
}

func openRange(ctx context.Context, backend backends.Backend, r *manifest.Range, offset int64) (io.ReadCloser, error) {
	if offset > r.Length {
		return nil, fmt.Errorf("invalid offset")
	}
	if offset == r.Length {
		return ioutil.NopCloser(bytes.NewReader(nil)), nil
	}
	rc, err := backend.Get(ctx, BlobPath(r.Blob), r.Offset+offset, r.Length-offset)
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
