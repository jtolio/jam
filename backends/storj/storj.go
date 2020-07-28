package storj

import (
	"context"
	"errors"
	"io"
	"net/url"
	"strings"

	"storj.io/common/socket"
	"storj.io/uplink"

	"github.com/jtolds/jam/backends"
	"github.com/zeebo/errs"
)

var (
	Error = errs.Class("storj error")
)

func init() {
	backends.Register("storj", New)
}

type Backend struct {
	p      *uplink.Project
	bucket string
	prefix string
}

func New(ctx context.Context, u *url.URL) (backends.Backend, error) {
	access, err := uplink.ParseAccess(u.Host)
	if err != nil {
		return nil, Error.Wrap(err)
	}

	p, err := (&uplink.Config{DialContext: socket.BackgroundDialer().DialContext}).OpenProject(ctx, access)
	if err != nil {
		return nil, Error.Wrap(err)
	}
	parts := strings.SplitN(strings.TrimPrefix(u.Path, "/"), "/", 2)
	var prefix string
	if len(parts) > 1 {
		prefix = parts[1]
	}

	return &Backend{
		p:      p,
		bucket: parts[0],
		prefix: prefix,
	}, nil
}

func (b *Backend) Get(ctx context.Context, path string, offset, length int64) (io.ReadCloser, error) {
	path = b.prefix + path
	d, err := b.p.DownloadObject(ctx, b.bucket, path, &uplink.DownloadOptions{Offset: offset, Length: length})
	if errors.Is(err, uplink.ErrObjectNotFound) {
		return d, Error.Wrap(backends.ErrNotExist)
	}
	return d, Error.Wrap(err)
}

func (b *Backend) Put(ctx context.Context, path string, data io.Reader) error {
	path = b.prefix + path
	u, err := b.p.UploadObject(ctx, b.bucket, path, nil)
	if err != nil {
		return Error.Wrap(err)
	}
	defer u.Abort()
	_, err = io.Copy(u, data)
	if err != nil {
		return Error.Wrap(err)
	}
	return Error.Wrap(u.Commit())
}

func (b *Backend) Delete(ctx context.Context, path string) error {
	path = b.prefix + path
	_, err := b.p.DeleteObject(ctx, b.bucket, path)
	if errors.Is(err, uplink.ErrObjectNotFound) {
		return nil
	}
	return Error.Wrap(err)
}

func (b *Backend) List(ctx context.Context, prefix string, cb func(ctx context.Context, path string) error) error {
	prefix = b.prefix + prefix
	it := b.p.ListObjects(ctx, b.bucket, &uplink.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	})
	for it.Next() {
		err := cb(ctx, strings.TrimPrefix(it.Item().Key, b.prefix))
		if err != nil {
			return err
		}
	}
	return Error.Wrap(it.Err())
}

func (b *Backend) Close() error {
	return Error.Wrap(b.p.Close())
}
