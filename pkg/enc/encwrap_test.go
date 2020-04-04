package enc

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/jtolds/jam/backends"
	"github.com/jtolds/jam/backends/backendtest"
	"github.com/jtolds/jam/backends/fs"
)

func TestFSBackend(t *testing.T) {
	backendtest.RunSuite(t, func() (backends.Backend, func() error, error) {
		td, err := ioutil.TempDir("", "fstest")
		if err != nil {
			return nil, nil, err
		}

		return NewEncWrapper(
				NewSecretboxCodec(16*1024),
				NewHMACKeyGenerator([]byte("hello")),
				fs.NewFS(td)),
			func() error {
				return os.RemoveAll(td)
			},
			nil
	})
}
