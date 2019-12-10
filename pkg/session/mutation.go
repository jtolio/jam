package session

import (
	"context"
	"io"
	"sync"
	"time"

	"github.com/jtolds/jam/backends"
)

func uploadBlob(ctx context.Context, backend backends.Backend, c *concat.Concatenator, blobSize int64) error {
	dest, err := c.Destination(ctx, blobSize)
	if err != nil {
		return err
	}
	name := idGen()
	err = backend.Put(ctx, stream.BlobPrefix+name, dest)
	if err != nil {
		dest.Close()
		return err
	}
	err = dest.Commit(name)
	if err != nil {
		dest.Close()
		return err
	}
	return dest.Close()
}

type Mutation struct {
	mtx sync.Mutex
	cv  *sync.Cond
}

func newMutation() *Mutation {
	m := &Mutation{}
	m.cv = sync.NewCond(&m.mtx)
	return m
}

func (m *Mutation) Delete(ctx context.Context, path string) error {
	panic("TODO")
}

func (m *Mutation) PutFile(ctx context.Context, path string, creation, modified time.Time, mode uint32, data io.Reader) error {
	panic("TODO")
}

func (m *Mutation) PutDir(ctx context.Context, path string, creation, modified time.Time, mode uint32) error {
	panic("TODO")
}

func (m *Mutation) PutSymlink(ctx context.Context, path string, creation, modified time.Time, mode uint32, target string) error {
	panic("TODO")
}

func (m *Mutation) Commit(ctx context.Context) error {
	panic("TODO")
}

func (m *Mutation) Abort(ctx context.Context) error {
	panic("TODO")
}
