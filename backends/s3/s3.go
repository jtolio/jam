package s3

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/zeebo/errs"

	"github.com/jtolds/jam/backends"
)

type Backend struct {
	bucket string
	svc    *s3.S3
}

func New(bucket string) (*Backend, error) {
	sess, err := session.NewSession()
	if err != nil {
		return nil, err
	}
	return &Backend{
		bucket: bucket,
		svc:    s3.New(sess),
	}, nil
}

var _ backends.Backend = (*Backend)(nil)

func (b *Backend) Get(ctx context.Context, path string, offset int64) (io.ReadCloser, error) {
	rangeOffset := fmt.Sprintf("%d-", offset)
	out, err := b.svc.GetObjectWithContext(ctx, &s3.GetObjectInput{
		Bucket: &b.bucket,
		Key:    &path,
		Range:  &rangeOffset,
	})
	if err != nil {
		return nil, err
	}
	return out.Body, nil
}

func (b *Backend) Put(ctx context.Context, path string, data io.Reader) error {
	// ugh, probably should spool to disk? this sucks, s3
	seekableData, err := ioutil.ReadAll(data)
	if err != nil {
		return err
	}
	_, err = b.svc.PutObjectWithContext(ctx, &s3.PutObjectInput{
		Body:   bytes.NewReader(seekableData),
		Bucket: &b.bucket,
		Key:    &path,
	})
	return err
}

func (b *Backend) Delete(ctx context.Context, path string) error {
	_, err := b.svc.DeleteObjectWithContext(ctx, &s3.DeleteObjectInput{
		Bucket: &b.bucket,
		Key:    &path,
	})
	return err
}

func (b *Backend) List(ctx context.Context, prefix string, cb func(ctx context.Context, path string) error) error {
	var internalErr error
	err := b.svc.ListObjectsPagesWithContext(ctx, &s3.ListObjectsInput{
		Bucket: &b.bucket,
		Prefix: &prefix,
	}, func(page *s3.ListObjectsOutput, lastPage bool) bool {
		for _, obj := range page.Contents {
			if internalErr != nil {
				return false
			}
			internalErr = cb(ctx, *obj.Key)
		}
		return internalErr == nil
	})
	return errs.Combine(err, internalErr)
}
