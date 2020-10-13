package s3

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
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

func New(ctx context.Context, u *url.URL) (backends.Backend, error) {
	parts := strings.SplitN(strings.TrimPrefix(u.Path, "/"), "/", 2)
	var prefix string
	bucket := parts[0]
	if len(parts) > 1 {
		prefix = parts[1]
	}

	accessKey := u.User.Username()
	secretKey, ok := u.User.Password()
	if !ok {
		return nil, Error.New("s3 url missing secret key. " +
			"format s3://<ak>:<sk>@<region/endpoint>/bucket/prefix?disable-ssl=false")
	}

	cfg := aws.NewConfig().
		WithCredentials(credentials.NewStaticCredentials(
			accessKey, secretKey, ""))

	if strings.Contains(u.Host, ".") {
		cfg = cfg.WithEndpoint(u.Host).
			WithRegion("us-east-1").
			WithS3ForcePathStyle(true)
	} else {
		cfg = cfg.WithRegion(u.Host)
	}

	switch strings.ToLower(u.Query().Get("disable-ssl")) {
	case "t", "y", "yes", "true":
		cfg = cfg.WithDisableSSL(true)
	}

	sess, err := session.NewSession(cfg)
	if err != nil {
		return nil, Error.Wrap(err)
	}
	return &Backend{
		bucket: bucket,
		prefix: prefix,
		svc:    s3.New(sess),
	}, nil
}

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
		if aerr, ok := err.(awserr.Error); ok && aerr.Code() == s3.ErrCodeNoSuchKey {
			return nil, Error.Wrap(backends.ErrNotExist)
		}
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
