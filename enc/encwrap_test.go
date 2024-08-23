package enc

import (
	"context"
	"net/url"
	"os"
	"testing"

	"github.com/jtolio/jam/backends"
	"github.com/jtolio/jam/backends/backendtest"
	"github.com/jtolio/jam/backends/fs"
	"github.com/jtolio/jam/hashdb"
)

var (
	ctx = context.Background()
)

func TestFSBackend(t *testing.T) {
	backendtest.RunSuite(t, func() (backends.Backend, func() error, error) {
		td, err := os.MkdirTemp("", "fstest")
		if err != nil {
			return nil, nil, err
		}

		b, err := fs.New(ctx, &url.URL{Path: td})
		if err != nil {
			return nil, nil, err
		}

		codecMap := NewCodecMap(NewSecretboxCodec(16 * 1024))
		codecMap.Register(hashdb.SmallHashsetSuffix,
			NewSecretboxCodec(1024))
		return NewEncWrapper(
				codecMap,
				NewHMACKeyGenerator([]byte("hello")),
				b),
			func() error {
				return os.RemoveAll(td)
			},
			nil
	})
}
