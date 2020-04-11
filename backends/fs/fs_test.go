package fs

import (
	"context"
	"io/ioutil"
	"net/url"
	"os"
	"testing"

	"github.com/jtolds/jam/backends"
	"github.com/jtolds/jam/backends/backendtest"
)

var (
	ctx = context.Background()
)

func TestFSBackend(t *testing.T) {
	backendtest.RunSuite(t, func() (backends.Backend, func() error, error) {
		td, err := ioutil.TempDir("", "fstest")
		if err != nil {
			return nil, nil, err
		}
		b, err := New(ctx, &url.URL{Path: td})
		if err != nil {
			return nil, nil, err
		}
		return b, func() error {
			return os.RemoveAll(td)
		}, nil
	})
}
