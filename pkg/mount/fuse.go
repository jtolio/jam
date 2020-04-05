package mount

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"
	"sync"
	"syscall"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"

	"github.com/jtolds/jam/pkg/manifest"
	"github.com/jtolds/jam/pkg/session"
	"github.com/jtolds/jam/pkg/streams"
)

type fuseHandle struct {
	mtx    sync.Mutex
	stream *streams.Stream
}

var _ fs.FileReader = (*fuseHandle)(nil)
var _ fs.FileReleaser = (*fuseHandle)(nil)

func (h *fuseHandle) Read(ctx context.Context, dest []byte, off int64) (fuse.ReadResult, syscall.Errno) {
	h.mtx.Lock()
	defer h.mtx.Unlock()
	_, err := h.stream.Seek(off, io.SeekStart)
	if err != nil {
		log.Printf("error: %+v", err)
		return nil, syscall.EIO
	}
	n, err := h.stream.Read(dest)
	if err == io.EOF {
		err = nil
	}
	if err != nil {
		log.Printf("error: %+v", err)
		return nil, syscall.EIO
	}
	return fuse.ReadResultData(dest[:n]), 0
}

func (h *fuseHandle) Release(ctx context.Context) syscall.Errno {
	h.mtx.Lock()
	defer h.mtx.Unlock()
	err := h.stream.Close()
	if err != nil {
		log.Printf("error: %+v", err)
	}
	return 0
}

type fuseNode struct {
	fs.Inode
	snap session.Snapshot
	path string
	meta *manifest.Metadata
	data *streams.Stream
}

var _ fs.InodeEmbedder = (*fuseNode)(nil)
var _ fs.NodeLookuper = (*fuseNode)(nil)
var _ fs.NodeReaddirer = (*fuseNode)(nil)
var _ fs.NodeOpener = (*fuseNode)(nil)
var _ fs.NodeReadlinker = (*fuseNode)(nil)
var _ fs.NodeGetattrer = (*fuseNode)(nil)

func fullpath(parent, name string) string {
	if parent == "" {
		return name
	}
	return parent + "/" + name
}

func (n *fuseNode) Lookup(ctx context.Context, name string, out *fuse.EntryOut) (
	*fs.Inode, syscall.Errno) {
	child := fullpath(n.path, name)
	meta, data, err := n.snap.Open(ctx, child)
	if err != nil {
		if errors.Is(err, session.ErrNotFound) {
			return n.NewInode(ctx, &fuseNode{
				snap: n.snap,
				path: child,
			}, fs.StableAttr{Mode: syscall.S_IFDIR}), 0
		}
		log.Printf("error: %+v", err)
		return nil, syscall.EIO
	}
	err = data.Close()
	if err != nil {
		log.Printf("error: %+v", err)
		return nil, syscall.EIO
	}

	out.Size = uint64(data.Length())
	out.Mtime = uint64(meta.Modified.Seconds)

	mode := uint32(syscall.S_IFREG)

	switch meta.Type {
	case manifest.Metadata_FILE:
	case manifest.Metadata_SYMLINK:
		mode = syscall.S_IFLNK
	default:
		log.Printf("unknown object type: %v", meta.Type)
		return nil, syscall.EIO
	}

	return n.NewInode(ctx, &fuseNode{
		snap: n.snap,
		path: child,
		meta: meta,
		data: data,
	}, fs.StableAttr{Mode: mode}), 0
}

func (n *fuseNode) Getattr(ctx context.Context, _ fs.FileHandle, out *fuse.AttrOut) syscall.Errno {
	if n.meta != nil {
		out.Mtime = uint64(n.meta.Modified.Seconds)
	}
	if n.data != nil {
		out.Size = uint64(n.data.Length())
	}
	return 0
}

func (n *fuseNode) Readdir(ctx context.Context) (fs.DirStream, syscall.Errno) {
	var entries []fuse.DirEntry
	err := n.snap.List(ctx, fullpath(n.path, ""), "/",
		func(ctx context.Context, entry *session.ListEntry) error {
			mode := uint32(fuse.S_IFDIR)
			if !entry.Prefix {
				switch entry.Meta.Type {
				case manifest.Metadata_FILE:
					mode = fuse.S_IFREG
				case manifest.Metadata_SYMLINK:
					mode = fuse.S_IFLNK
				default:
					return fmt.Errorf("unknown object type: %v", entry.Meta.Type)
				}
			}
			entries = append(entries, fuse.DirEntry{
				Mode: mode,
				Name: strings.TrimPrefix(entry.Path, fullpath(n.path, "")),
			})
			return nil
		})
	if err != nil {
		log.Printf("error: %+v", err)
		return nil, syscall.EIO
	}
	return fs.NewListDirStream(entries), 0
}

func (n *fuseNode) Open(ctx context.Context, openFlags uint32) (
	fh fs.FileHandle, fuseFlags uint32, errno syscall.Errno) {
	return &fuseHandle{stream: n.data.Fork(ctx)}, fuse.FOPEN_KEEP_CACHE, 0
}

func (n *fuseNode) Readlink(ctx context.Context) ([]byte, syscall.Errno) {
	return []byte(n.meta.LinkTarget), 0
}

type Session struct {
	server *fuse.Server
}

func Mount(ctx context.Context, snap session.Snapshot, target string) (*Session, error) {
	server, err := fs.Mount(target, &fuseNode{snap: snap}, &fs.Options{
		MountOptions: fuse.MountOptions{
			DisableXAttrs: true,
		}})
	if err != nil {
		return nil, err
	}
	err = server.WaitMount()
	if err != nil {
		server.Unmount()
		return nil, err
	}
	return &Session{server: server}, nil
}

func (s *Session) Wait() {
	s.server.Serve()
}

func (s *Session) Close() error {
	return s.server.Unmount()
}
