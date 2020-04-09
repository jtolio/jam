// +build ignore

package storj

import (
	"context"
	"fmt"
	"testing"
	"time"

	"storj.io/uplink"

	"github.com/jtolds/jam/backends"
	"github.com/jtolds/jam/backends/backendtest"
)

var (
	ctx         = context.Background()
	accessGrant = "<fillin>"
)

func TestStorjBackend(t *testing.T) {
	backendtest.RunSuite(t, func() (backends.Backend, func() error, error) {
		access, err := uplink.ParseAccess(ctx, accessGrant)
		if err != nil {
			return nil, nil, err
		}
		p, err := uplink.OpenProject(ctx, access)
		if err != nil {
			return nil, nil, err
		}
		bucket := fmt.Sprintf("bucket-%d", time.Now().UnixNano())
		_, err = p.CreateBucket(ctx, bucket)
		if err != nil {
			p.Close()
			return nil, nil, err
		}
		return New(p, bucket), func() error {
			return p.Close()
		}, nil
	})
}
