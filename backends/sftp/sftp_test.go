//go:build ignore
// +build ignore

package sftp

import (
	"context"
	"net/url"
	"testing"

	"github.com/jtolio/jam/backends"
	"github.com/jtolio/jam/backends/backendtest"
	"github.com/jtolio/jam/blobs"
)

var (
	ctx            = context.Background()
	testHost       = "thinclient.lan"
	testUser       = "jt"
	testPathPrefix = "/mnt/data/jt/testing/"
)

func TestSFTPBackend(t *testing.T) {
	backendtest.RunSuite(t, func() (backends.Backend, func() error, error) {
		test := blobs.IdGen()
		b, err := New(ctx, &url.URL{
			Host: testHost,
			User: url.User(testUser),
			Path: testPathPrefix + test + "/"})
		return b, nil, err
	})
}
