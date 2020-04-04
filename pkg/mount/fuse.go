package mount

import (
	"context"
	"syscall"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"

	"github.com/jtolds/jam/pkg/session"
)

type fuseNode struct {
	fs.Inode
}

var _ fs.InodeEmbedder = (*fuseNode)(nil)
var _ fs.NodeLookuper = (*fuseNode)(nil)

func (n *fuseNode) Lookup(ctx context.Context, name string, out *fuse.EntryOut) (
	*fs.Inode, syscall.Errno) {
	return n.NewInode(ctx, &fuseNode{}, fs.StableAttr{Mode: syscall.S_IFDIR}), 0
}

func Mount(ctx context.Context, snap session.Snapshot, target string) error {
	server, err := fs.Mount(target, &fuseNode{}, nil)
	if err != nil {
		return err
	}
	server.Serve()
	return nil
}
