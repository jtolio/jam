package fs

import (
	"context"
	"io"
	"io/ioutil"
	"math/rand"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/natefinch/atomic"
	"github.com/zeebo/errs"

	"github.com/jtolds/jam/backends"
)

func init() {
	backends.Register("file", New)
}

// FS implements the Backend interface using the local disk
type FS struct {
	root string
}

// New returns an FS mounted at the provided root path.
func New(ctx context.Context, u *url.URL) (backends.Backend, error) {
	return &FS{root: u.Path}, nil
}

// Get implements the Backend interface
func (fs *FS) Get(ctx context.Context, path string, offset, length int64) (rv io.ReadCloser, err error) {
	localpath := filepath.Join(fs.root, path)
	fh, err := os.Open(localpath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errs.Wrap(backends.ErrNotExist)
		}
		return nil, errs.Wrap(err)
	}
	if offset > 0 {
		_, err = fh.Seek(offset, io.SeekStart)
		if err != nil {
			fh.Close()
			return nil, errs.Wrap(err)
		}
	}

	rv = fh
	if length > 0 && rand.Intn(2) == 0 {
		// makes sure we make the rest of the code handle both cases, since other backends might do
		// either thing and this backend is used often for testing
		rv = struct {
			io.Reader
			io.Closer
		}{
			Reader: io.LimitReader(fh, length),
			Closer: fh,
		}
	}

	return rv, nil
}

// Put implements the Backend interface
func (fs *FS) Put(ctx context.Context, path string, data io.Reader) (err error) {
	defer func() {
		if err != nil {
			fs.Delete(ctx, path)
		}
	}()

	localpath := filepath.Join(fs.root, path)
	err = os.MkdirAll(filepath.Dir(localpath), 0700)
	if err != nil {
		return errs.Wrap(err)
	}

	dir, file := filepath.Split(localpath)
	fh, err := ioutil.TempFile(dir, "_"+file)
	if err != nil {
		return errs.Wrap(err)
	}
	defer fh.Close()

	name := fh.Name()
	defer func() {
		if err != nil {
			_ = os.Remove(name)
		}
	}()

	_, err = io.Copy(fh, data)
	if err != nil {
		return errs.Wrap(err)
	}

	err = fh.Sync()
	if err != nil {
		return errs.Wrap(err)
	}

	err = fh.Close()
	if err != nil {
		return errs.Wrap(err)
	}

	return errs.Wrap(atomic.ReplaceFile(name, localpath))
}

// Delete implements the Backend interface
func (fs *FS) Delete(ctx context.Context, path string) error {
	localpath := filepath.Join(fs.root, path)
	if _, err := os.Lstat(localpath); os.IsNotExist(err) {
		return nil
	}
	err := os.Remove(localpath)
	if err != nil {
		return errs.Wrap(err)
	}
	// the rest is not required but is an attempt to be nice and clean up intermediate
	// directories after ourselves. remove any parents up to the root that are empty
	for {
		localpath = filepath.Dir(localpath)
		rel, err := filepath.Rel(fs.root, localpath)
		if err != nil || rel == "." {
			return nil
		}
		err = os.Remove(localpath)
		if err != nil {
			return nil
		}
	}
}

// List implements the Backend interface
func (fs *FS) List(ctx context.Context, prefix string,
	cb func(ctx context.Context, path string) error) error {
	localpath := filepath.Join(fs.root, prefix)
	if s, err := os.Lstat(localpath); os.IsNotExist(err) || !s.IsDir() {
		return nil
	}
	return filepath.Walk(filepath.Join(fs.root, prefix),
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return errs.Wrap(err)
			}
			if !info.Mode().IsRegular() {
				return nil
			}

			if strings.HasPrefix(filepath.Base(path), "_") {
				return nil
			}

			internal, err := filepath.Rel(fs.root, path)
			if err != nil {
				return errs.Wrap(err)
			}

			return cb(ctx, internal)
		})
}

func (fs *FS) Close() error { return nil }
