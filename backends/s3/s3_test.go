// +build ignore

package s3

import (
	"context"
	"testing"

	"github.com/jtolds/jam/backends"
	"github.com/jtolds/jam/backends/backendtest"
)

var (
	ctx    = context.Background()
	bucket = "<fillin>"
)

func TestStorjBackend(t *testing.T) {
	backendtest.RunSuite(t, func() (backends.Backend, func() error, error) {
		b, err := New(bucket)
		return b, func() error {
			return nil
		}, err
	})
}
