package mount

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"syscall"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/golang/protobuf/ptypes"
	"github.com/zeebo/errs"

	"github.com/jtolds/jam/manifest"
	"github.com/jtolds/jam/session"
	"github.com/jtolds/jam/streams"
)

type fuseHandle struct {
	mtx    sync.Mutex
	stream *streams.Stream
}

var _ fs.HandleReader = (*fuseHandle)(nil)
var _ fs.HandleReleaser = (*fuseHandle)(nil)

func logE(err error) error {
	if err != nil {
		log.Printf("error: %+v\n", err)
	}
	return err
}

func (h *fuseHandle) Read(ctx context.Context, req *fuse.ReadRequest, resp *fuse.ReadResponse) error {
	h.mtx.Lock()
	defer h.mtx.Unlock()
	_, err := h.stream.Seek(req.Offset, io.SeekStart)
	if err != nil {
		return logE(err)
	}
	dest := resp.Data
	if len(dest) < req.Size {
		dest = make([]byte, req.Size)
	}
	n, err := io.ReadFull(h.stream, dest)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return logE(err)
	}
	resp.Data = dest[:n]
	return nil
}

func (h *fuseHandle) Release(ctx context.Context, req *fuse.ReleaseRequest) error {
	h.mtx.Lock()
	defer h.mtx.Unlock()
	return logE(h.stream.Close())
}

type fuseNode struct {
	snap *session.Snapshot
	path string
	meta *manifest.Metadata
	data *streams.Stream
}

var _ fs.NodeStringLookuper = (*fuseNode)(nil)
var _ fs.NodeOpener = (*fuseNode)(nil)
var _ fs.NodeReadlinker = (*fuseNode)(nil)

func fullpath(parent, name string) string {
	if parent == "" {
		return name
	}
	return parent + "/" + name
}

func (n *fuseNode) Lookup(ctx context.Context, name string) (fs.Node, error) {
	child := fullpath(n.path, name)
	meta, data, err := n.snap.Open(ctx, child)
	if err != nil {
		if !errors.Is(err, session.ErrNotFound) {
			return nil, logE(err)
		}
		exists, err := n.snap.HasPrefix(ctx, fullpath(child, ""))
		if err != nil {
			return nil, logE(err)
		}
		if !exists {
			return nil, fuse.ENOENT
		}
		meta, data = nil, nil
	} else if data != nil {
		err = data.Close()
		if err != nil {
			return nil, logE(err)
		}
	}
	return &fuseNode{
		snap: n.snap,
		path: child,
		meta: meta,
		data: data,
	}, nil
}

var (
	uid = uint32(os.Geteuid())
	gid = uint32(os.Getegid())
)

func (n *fuseNode) Attr(ctx context.Context, out *fuse.Attr) error {
	out.Uid = uid
	out.Gid = gid
	if n.meta == nil {
		out.Mode = os.ModeDir | 0500
		return nil
	}
	modTime, err := ptypes.Timestamp(n.meta.Modified)
	if err != nil {
		return logE(err)
	}
	out.Mtime = modTime
	out.Ctime = modTime
	crTime, err := ptypes.Timestamp(n.meta.Creation)
	if err != nil {
		return logE(err)
	}
	out.Crtime = crTime
	if n.data != nil {
		out.Size = uint64(n.data.Length())
		out.Blocks = out.Size / 512
		if out.Size%512 != 0 {
			out.Blocks++
		}
	}
	switch n.meta.Type {
	case manifest.Metadata_FILE:
	case manifest.Metadata_SYMLINK:
		out.Mode = os.ModeSymlink
	default:
		return logE(fmt.Errorf("unknown object type: %v", n.meta.Type))
	}
	out.Mode |= os.FileMode(n.meta.Mode & 0777)
	return nil
}

type fuseDir struct {
	n *fuseNode
}

var _ fs.HandleReadDirAller = (*fuseDir)(nil)

func (d *fuseDir) ReadDirAll(ctx context.Context) (entries []fuse.Dirent, err error) {
	prefix := fullpath(d.n.path, "")
	err = d.n.snap.List(ctx, prefix, "/",
		func(ctx context.Context, entry *session.ListEntry) error {
			mode := fuse.DT_Dir
			if !entry.Prefix {
				switch entry.Meta.Type {
				case manifest.Metadata_FILE:
					mode = fuse.DT_File
				case manifest.Metadata_SYMLINK:
					mode = fuse.DT_Link
				default:
					return fmt.Errorf("unknown object type: %v", entry.Meta.Type)
				}
			}
			entries = append(entries, fuse.Dirent{
				Type: mode,
				Name: strings.TrimPrefix(entry.Path, prefix),
			})
			return nil
		})
	return entries, logE(err)
}

func (n *fuseNode) Open(ctx context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (
	fs.Handle, error) {
	if !req.Flags.IsReadOnly() {
		return nil, fuse.Errno(syscall.EACCES)
	}
	resp.Flags |= fuse.OpenKeepCache
	if req.Dir {
		return &fuseDir{n: n}, nil
	}
	return &fuseHandle{stream: n.data.Fork(context.Background())}, nil
}

func (n *fuseNode) Readlink(ctx context.Context, req *fuse.ReadlinkRequest) (string, error) {
	return n.meta.LinkTarget, nil
}

type fuseFS struct {
	root *fuseNode
}

func (f *fuseFS) Root() (fs.Node, error) { return f.root, nil }

type Session struct {
	conn   *fuse.Conn
	target string
	srv    *fs.Server
	fs     *fuseFS
}

func Mount(ctx context.Context, snap *session.Snapshot, target string) (*Session, error) {
	conn, err := fuse.Mount(target, fuse.FSName("jam"), fuse.ReadOnly())
	if err != nil {
		return nil, err
	}
	return &Session{
		conn:   conn,
		target: target,
		srv:    fs.New(conn, nil),
		fs:     &fuseFS{root: &fuseNode{snap: snap}},
	}, nil
}

func (s *Session) Serve() error {
	err := s.srv.Serve(s.fs)
	if err != nil {
		return err
	}
	<-s.conn.Ready
	return s.conn.MountError
}

func (s *Session) Close() error {
	err := fuse.Unmount(s.target)
	return errs.Combine(err, s.conn.Close())
}
