package webdav

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/zeebo/errs"
	"golang.org/x/net/webdav"

	"github.com/jtolds/jam/manifest"
	"github.com/jtolds/jam/session"
	"github.com/jtolds/jam/streams"
)

var (
	ErrReadOnly = errs.New("read only")
)

type webdavFS struct {
	snap *session.Snapshot
}

func WebdavFS(snap *session.Snapshot) webdav.FileSystem {
	return &webdavFS{snap: snap}
}

func (fs *webdavFS) Mkdir(ctx context.Context, name string, perm os.FileMode) error {
	return ErrReadOnly
}

func (fs *webdavFS) RemoveAll(ctx context.Context, name string) error {
	return ErrReadOnly
}

func (fs *webdavFS) Rename(ctx context.Context, oldname, newname string) error {
	return ErrReadOnly
}

func (fs *webdavFS) Stat(ctx context.Context, name string) (os.FileInfo, error) {
	fh, err := fs.OpenFile(ctx, name, os.O_RDONLY, 0)
	if err != nil {
		return nil, err
	}
	defer fh.Close()
	return fh.Stat()
}

type fileInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
}

func (f *fileInfo) Name() string       { return f.name }
func (f *fileInfo) Size() int64        { return f.size }
func (f *fileInfo) Mode() os.FileMode  { return f.mode }
func (f *fileInfo) ModTime() time.Time { return f.modTime }
func (f *fileInfo) IsDir() bool        { return f.mode.IsDir() }
func (f *fileInfo) Sys() interface{}   { return nil }

type webdavDir struct {
	self    fs.FileInfo
	entries []fs.FileInfo
}

func (d *webdavDir) Close() error                                 { return nil }
func (d *webdavDir) Read(p []byte) (n int, err error)             { return 0, io.EOF }
func (d *webdavDir) Seek(offset int64, whence int) (int64, error) { return 0, nil }

func (d *webdavDir) Readdir(count int) (entries []fs.FileInfo, err error) {
	max := count
	if max <= 0 || max > len(d.entries) {
		max = len(d.entries)
	}
	entries, d.entries = d.entries[:max], d.entries[max:]
	if len(entries) == 0 && count > 0 {
		return nil, io.EOF
	}
	return entries, nil
}

func (d *webdavDir) Stat() (fs.FileInfo, error)        { return d.self, nil }
func (d *webdavDir) Write(p []byte) (n int, err error) { return 0, ErrReadOnly }

type webdavFile struct {
	snap *session.Snapshot
	path string
	meta *manifest.Metadata
	data *streams.Stream
}

func calcMode(meta *manifest.Metadata) (os.FileMode, error) {
	var mode os.FileMode
	switch meta.Type {
	case manifest.Metadata_FILE:
		mode = 0
	case manifest.Metadata_SYMLINK:
		mode = os.ModeSymlink
	default:
		return 0, fmt.Errorf("unknown object type: %v", meta.Type)
	}
	return mode | os.FileMode(meta.Mode&0777), nil
}

func (f *webdavFile) Close() error                     { return f.data.Close() }
func (f *webdavFile) Read(p []byte) (n int, err error) { return f.data.Read(p) }
func (f *webdavFile) Seek(offset int64, whence int) (int64, error) {
	return f.data.Seek(offset, whence)
}
func (f *webdavFile) Readdir(count int) ([]fs.FileInfo, error) { return nil, nil }
func (f *webdavFile) Stat() (fs.FileInfo, error) {
	mode, err := calcMode(f.meta)
	if err != nil {
		return nil, err
	}
	modTime, err := ptypes.Timestamp(f.meta.Modified)
	if err != nil {
		return nil, err
	}
	return &fileInfo{
		name:    filepath.Base(f.path),
		size:    f.data.Length(),
		mode:    mode,
		modTime: modTime,
	}, nil
}
func (f *webdavFile) Write(p []byte) (n int, err error) { return 0, ErrReadOnly }

func (fs *webdavFS) OpenFile(ctx context.Context, name string, flag int, perm os.FileMode) (webdav.File, error) {
	name = strings.TrimPrefix(name, "/")
	meta, data, err := fs.snap.Open(ctx, name)
	if err == nil {
		return &webdavFile{snap: fs.snap, path: name, meta: meta, data: data}, nil
	}
	if !errors.Is(err, session.ErrNotFound) {
		return nil, err
	}
	pre := name
	if len(pre) > 0 && pre[len(pre)-1:] != "/" {
		pre += "/"
	}
	var entries []os.FileInfo
	err = fs.snap.List(ctx, pre, false,
		func(ctx context.Context, entry *session.ListEntry) error {
			mode := os.ModeDir | 0555
			size := int64(0)
			var modTime time.Time
			if !entry.Prefix {
				switch entry.Meta.Type {
				case manifest.Metadata_FILE:
					mode = 0
					str, err := entry.Stream(ctx)
					if err != nil {
						return err
					}
					size = str.Length()
					err = str.Close()
					if err != nil {
						return err
					}
				case manifest.Metadata_SYMLINK:
					mode = os.ModeSymlink
				default:
					return fmt.Errorf("unknown object type: %v", entry.Meta.Type)
				}
				mode |= os.FileMode(entry.Meta.Mode & 0777)
				var err error
				modTime, err = ptypes.Timestamp(entry.Meta.Modified)
				if err != nil {
					return err
				}
			}
			entries = append(entries, &fileInfo{
				name:    strings.TrimPrefix(entry.Path, pre),
				size:    size,
				mode:    mode,
				modTime: modTime,
			})
			return nil
		})
	if len(entries) == 0 {
		return nil, os.ErrNotExist
	}
	return &webdavDir{
		self: &fileInfo{
			name: filepath.Base(name),
			mode: os.ModeDir,
		},
		entries: entries,
	}, nil
}

func Serve(ctx context.Context, snap *session.Snapshot, addr string) error {
	handler := &webdav.Handler{
		FileSystem: WebdavFS(snap),
		LockSystem: webdav.NewMemLS(),
	}
	return http.ListenAndServe(addr, handler)
}
