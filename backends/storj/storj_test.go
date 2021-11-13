//go:build ignore
// +build ignore

package storj

import (
	"context"
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/jtolio/jam/backends"
	"github.com/jtolio/jam/backends/backendtest"
)

var (
	ctx         = context.Background()
	accessGrant = "<fillin>"
)

func TestStorjBackend(t *testing.T) {
	backendtest.RunSuite(t, func() (backends.Backend, func() error, error) {
		bucket := fmt.Sprintf("jam-test-bucket-%d", time.Now().UnixNano())
		b, err := New(ctx, &url.URL{Host: accessGrant, Path: "/" + bucket})
		if err != nil {
			return nil, nil, err
		}
		_, err = b.(*Backend).p.CreateBucket(ctx, bucket)
		if err != nil {
			b.Close()
			return nil, nil, err
		}
		return b, nil, nil
	})
}

func TestStorjBackendWithPrefix(t *testing.T) {
	backendtest.RunSuite(t, func() (backends.Backend, func() error, error) {
		bucket := fmt.Sprintf("jam-test-bucket-%d", time.Now().UnixNano())
		b, err := New(ctx, &url.URL{Host: accessGrant, Path: "/" + bucket + "/aprefix/"})
		if err != nil {
			return nil, nil, err
		}
		_, err = b.(*Backend).p.CreateBucket(ctx, bucket)
		if err != nil {
			b.Close()
			return nil, nil, err
		}
		return b, nil, nil
	})
}
