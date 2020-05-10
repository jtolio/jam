package s3

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"strings"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/zeebo/errs"

	"github.com/jtolds/jam/backends"
)

var (
	Error = errs.Class("s3 error")
)

func init() {
	backends.Register("s3", New)
}

type Backend struct {
	bucket string
	prefix string
	svc    *s3.S3
}

func New(ctx context.Context, url *url.URL) (backends.Backend, error) {
	sess, err := session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable})
	if err != nil {
		return nil, Error.Wrap(err)
	}
	return &Backend{
		bucket: url.Host,
		prefix: strings.TrimPrefix(url.Path, "/"),
		svc:    s3.New(sess),
	}, nil
}

var _ backends.Backend = (*Backend)(nil)

func (b *Backend) Get(ctx context.Context, path string, offset, length int64) (io.ReadCloser, error) {
	rangeOffset := fmt.Sprintf("bytes=%d-", offset)
	if length > 0 {
		rangeOffset = fmt.Sprintf("bytes=%d-%d", offset, offset+length-1)
	}
	path = b.prefix + path
	out, err := b.svc.GetObjectWithContext(ctx, &s3.GetObjectInput{
		Bucket: &b.bucket,
		Key:    &path,
		Range:  &rangeOffset,
	})
	if err != nil {
		return nil, Error.Wrap(err)
	}
	return out.Body, nil
}

func (b *Backend) Put(ctx context.Context, path string, data io.Reader) error {
	// ugh, probably should spool to disk? this sucks, s3
	seekableData, err := ioutil.ReadAll(data)
	if err != nil {
		return Error.Wrap(err)
	}
	path = b.prefix + path
	_, err = b.svc.PutObjectWithContext(ctx, &s3.PutObjectInput{
		Body:   bytes.NewReader(seekableData),
		Bucket: &b.bucket,
		Key:    &path,
	})
	return Error.Wrap(err)
}

func (b *Backend) Delete(ctx context.Context, path string) error {
	path = b.prefix + path
	_, err := b.svc.DeleteObjectWithContext(ctx, &s3.DeleteObjectInput{
		Bucket: &b.bucket,
		Key:    &path,
	})
	return Error.Wrap(err)
}

func (b *Backend) List(ctx context.Context, prefix string, cb func(ctx context.Context, path string) error) error {
	var internalErr error
	prefix = b.prefix + prefix
	err := b.svc.ListObjectsPagesWithContext(ctx, &s3.ListObjectsInput{
		Bucket: &b.bucket,
		Prefix: &prefix,
	}, func(page *s3.ListObjectsOutput, lastPage bool) bool {
		for _, obj := range page.Contents {
			if internalErr != nil {
				return false
			}
			internalErr = cb(ctx, strings.TrimPrefix(*obj.Key, b.prefix))
		}
		return internalErr == nil
	})
	return Error.Wrap(errs.Combine(err, internalErr))
}

func (b *Backend) Close() error { return nil }
