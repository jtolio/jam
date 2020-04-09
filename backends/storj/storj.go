package storj

import (
	"context"
	"io"

	"storj.io/uplink"

	"github.com/jtolds/jam/backends"
)

type Backend struct {
	p      *uplink.Project
	bucket string
}

func New(p *uplink.Project, bucket string) *Backend {
	return &Backend{
		p:      p,
		bucket: bucket,
	}
}

var _ backends.Backend = (*Backend)(nil)

func (b *Backend) Get(ctx context.Context, path string, offset int64) (io.ReadCloser, error) {
	d, err := b.p.DownloadObject(ctx, b.bucket, path, &uplink.DownloadOptions{Offset: offset, Length: -1})
	return d, err
}

func (b *Backend) Put(ctx context.Context, path string, data io.Reader) error {
	u, err := b.p.UploadObject(ctx, b.bucket, path, nil)
	if err != nil {
		return err
	}
	defer u.Abort()
	_, err = io.Copy(u, data)
	if err != nil {
		return err
	}
	return u.Commit()
}

func (b *Backend) Delete(ctx context.Context, path string) error {
	_, err := b.p.DeleteObject(ctx, b.bucket, path)
	return err
}

func (b *Backend) List(ctx context.Context, prefix string, cb func(ctx context.Context, path string) error) error {
	it := b.p.ListObjects(ctx, b.bucket, &uplink.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	})
	for it.Next() {
		err := cb(ctx, it.Item().Key)
		if err != nil {
			return err
		}
	}
	return it.Err()
}
