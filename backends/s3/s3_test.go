// +build ignore

package s3

import (
	"context"
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/service/s3"

	"github.com/jtolds/jam/backends"
	"github.com/jtolds/jam/backends/backendtest"
)

var (
	ctx = context.Background()
)

func TestS3Backend(t *testing.T) {
	backendtest.RunSuite(t, func() (backends.Backend, func() error, error) {
		bucket := fmt.Sprintf("jam-test-bucket-%d", time.Now().UnixNano())
		b, err := New(ctx, &url.URL{Host: bucket})
		if err != nil {
			return nil, nil, err
		}
		_, err = b.(*Backend).svc.CreateBucket(&s3.CreateBucketInput{
			Bucket: &bucket,
		})
		if err != nil {
			b.Close()
			return nil, nil, err
		}
		return b, nil, err
	})
}

func TestS3BackendWithPrefix(t *testing.T) {
	backendtest.RunSuite(t, func() (backends.Backend, func() error, error) {
		bucket := fmt.Sprintf("jam-test-bucket-%d", time.Now().UnixNano())
		b, err := New(ctx, &url.URL{Host: bucket, Path: "/aprefix/"})
		if err != nil {
			return nil, nil, err
		}
		_, err = b.(*Backend).svc.CreateBucket(&s3.CreateBucketInput{
			Bucket: &bucket,
		})
		if err != nil {
			b.Close()
			return nil, nil, err
		}
		return b, nil, nil
	})
}
