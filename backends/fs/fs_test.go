package fs

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/jtolds/jam/backends"
	"github.com/jtolds/jam/backends/backendtest"
)

func TestFSBackend(t *testing.T) {
	backendtest.RunSuite(t, func() (backends.Backend, func() error, error) {
		td, err := ioutil.TempDir("", "fstest")
		if err != nil {
			return nil, nil, err
		}

		return NewFS(td), func() error {
			return os.RemoveAll(td)
		}, nil
	})
}
