package fs

import (
	"context"
	"io"
	"os"
	"path/filepath"

	"github.com/zeebo/errs"

	"github.com/jtolds/jam/backends"
)

// FS implements the Backend interface using the local disk
type FS struct {
	root string
}

var _ backends.Backend = (*FS)(nil)

// NewFS returns an FS mounted at the provided root path.
func NewFS(root string) *FS {
	return &FS{root: root}
}

// Get implements the Backend interface
func (fs *FS) Get(ctx context.Context, path string, offset int64) (io.ReadCloser, error) {
	localpath := filepath.Join(fs.root, path)
	fh, err := os.Open(localpath)
	if err != nil {
		return nil, errs.Wrap(err)
	}
	if offset > 0 {
		_, err = fh.Seek(offset, io.SeekStart)
		if err != nil {
			fh.Close()
			return nil, errs.Wrap(err)
		}
	}
	return fh, nil
}

// Put implements the Backend interface
func (fs *FS) Put(ctx context.Context, path string, data io.Reader) error {
	localpath := filepath.Join(fs.root, path)
	err := os.MkdirAll(filepath.Dir(localpath), 0700)
	if err != nil {
		return errs.Wrap(err)
	}

	fh, err := os.Create(localpath)
	if err != nil {
		return err
	}

	_, err = io.Copy(fh, data)
	if err != nil {
		fh.Close()
		return errs.Wrap(err)
	}

	return errs.Wrap(fh.Close())
}

// Delete implements the Backend interface
func (fs *FS) Delete(ctx context.Context, path string) error {
	localpath := filepath.Join(fs.root, path)
	if _, err := os.Stat(localpath); os.IsNotExist(err) {
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
	if s, err := os.Stat(localpath); os.IsNotExist(err) || !s.IsDir() {
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
			internal, err := filepath.Rel(fs.root, path)
			if err != nil {
				return errs.Wrap(err)
			}
			return cb(ctx, internal)
		})
}
