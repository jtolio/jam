package session

import (
	"context"
	"io"
	"sync"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/jtolds/jam/backends"
	"github.com/jtolds/jam/pkg/concat"
	"github.com/jtolds/jam/pkg/manifest"
	"github.com/jtolds/jam/pkg/stream"
)

type Mutation struct {
	backend  backends.Backend
	blobSize int64
	concat   *concat.Concatenator

	mtx    sync.Mutex
	cv     *sync.Cond
	closed bool
	err    error
}

func newMutation(ctx context.Context, backend backends.Backend, blobSize int64) *Mutation {
	m := &Mutation{
		backend:  backend,
		blobSize: blobSize,
		concat:   concat.NewConcatenator(),
	}
	m.cv = sync.NewCond(&m.mtx)
	go m.background(ctx)
	return m
}

func (m *Mutation) background(ctx context.Context) {
	for {
		err := m.uploadBlob(ctx)
		m.mtx.Lock()
		if m.closed {
			m.mtx.Unlock()
			return
		}
		if err != nil {
			m.closeLocked(err)
			m.mtx.Unlock()
			return
		}
		m.mtx.Unlock()
	}
}

func (m *Mutation) uploadBlob(ctx context.Context) error {
	dest, err := m.concat.Destination(ctx, m.blobSize)
	if err != nil {
		return err
	}
	name := idGen()
	err = m.backend.Put(ctx, stream.BlobPrefix+name, dest)
	if err != nil {
		dest.Abort()
		return err
	}
	return dest.Commit(name)
}

func (m *Mutation) closeLocked(err error) {
	if m.err == nil {
		m.err = err
	}
	err = m.concat.Close()
	if m.err == nil {
		m.err = err
	}
	m.closed = true
}

func (m *Mutation) Delete(ctx context.Context, path string) error {
	panic("TODO")
}

func (m *Mutation) PutFile(ctx context.Context, path string, creation, modified time.Time, mode uint32, data io.Reader) error {
	stream, err := m.concat.Add(ctx, data)
	if err != nil {
		return err
	}

	creationPB, modifiedPB, err := convertTime(creation, modified)
	if err != nil {
		return err
	}

	return m.put(ctx, path, &manifest.Content{
		Metadata: &manifest.Metadata{
			Type:     manifest.Metadata_FILE,
			Creation: creationPB,
			Modified: modifiedPB,
			Mode:     mode,
		},
		Data: stream,
	})
}

func (m *Mutation) PutDir(ctx context.Context, path string, creation, modified time.Time, mode uint32) error {
	creationPB, modifiedPB, err := convertTime(creation, modified)
	if err != nil {
		return err
	}
	return m.put(ctx, path, &manifest.Content{
		Metadata: &manifest.Metadata{
			Type:     manifest.Metadata_DIR,
			Creation: creationPB,
			Modified: modifiedPB,
			Mode:     mode,
		}})
}

func (m *Mutation) PutSymlink(ctx context.Context, path string, creation, modified time.Time, mode uint32, target string) error {
	creationPB, modifiedPB, err := convertTime(creation, modified)
	if err != nil {
		return err
	}
	return m.put(ctx, path, &manifest.Content{
		Metadata: &manifest.Metadata{
			Type:       manifest.Metadata_SYMLINK,
			Creation:   creationPB,
			Modified:   modifiedPB,
			Mode:       mode,
			LinkTarget: target,
		}})
}

func (m *Mutation) put(ctx context.Context, path string, content *manifest.Content) error {
	panic("TODO")
}

func convertTime(a, b time.Time) (*timestamp.Timestamp, *timestamp.Timestamp, error) {
	apb, err := ptypes.TimestampProto(a)
	if err != nil {
		return nil, nil, err
	}
	bpb, err := ptypes.TimestampProto(b)
	if err != nil {
		return nil, nil, err
	}
	return apb, bpb, nil
}

func (m *Mutation) List(ctx context.Context, prefix string, cb func(context.Context, manifest.Entry) error) error {
	panic("TODO")
}

func (m *Mutation) Open(ctx context.Context, path string) (*manifest.Metadata, *stream.Stream, error) {
	panic("TODO")
}

func (m *Mutation) Commit(ctx context.Context) error {
	err := m.concat.Commit(ctx)
	if err != nil {
		return err
	}
	// TODO shut down goroutine
	panic("TODO")
}

func (m *Mutation) Abort(ctx context.Context) error {
	panic("TODO")
}
