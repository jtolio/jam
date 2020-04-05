package mount

import (
	"context"
	"errors"
	"log"
	"strings"
	"syscall"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"

	"github.com/jtolds/jam/pkg/manifest"
	"github.com/jtolds/jam/pkg/session"
	"github.com/jtolds/jam/pkg/streams"
)

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

func (n *fuseNode) Lookup(ctx context.Context, name string, out *fuse.EntryOut) (
	*fs.Inode, syscall.Errno) {
	child := n.path + "/" + name
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

	return n.NewInode(ctx, &fuseNode{
		snap: n.snap,
		path: child,
		meta: meta,
		data: data,
	}, fs.StableAttr{Mode: syscall.S_IFREG}), 0 // TODO: symlinks
}

func (n *fuseNode) Readdir(ctx context.Context) (fs.DirStream, syscall.Errno) {
	var entries []fuse.DirEntry
	err := n.snap.List(ctx, n.path+"/", "/",
		func(ctx context.Context, entry *session.ListEntry) error {
			mode := uint32(fuse.S_IFREG)
			if entry.Prefix {
				mode = fuse.S_IFDIR
			}
			// TODO: symlinks
			entries = append(entries, fuse.DirEntry{
				Mode: mode,
				Name: strings.TrimPrefix(entry.Path, n.path+"/"),
			})
			return nil
		})
	if err != nil {
		log.Printf("error: %+v", err)
		return nil, syscall.EIO
	}
	return fs.NewListDirStream(entries), 0
}

func Mount(ctx context.Context, snap session.Snapshot, target string) error {
	server, err := fs.Mount(target, &fuseNode{snap: snap}, nil)
	if err != nil {
		return err
	}
	server.Serve()
	return nil
}
