package session

import (
	"context"
	"io"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/jtolds/jam/backends"
	"github.com/jtolds/jam/pkg/manifest"
	"github.com/jtolds/jam/pkg/stream"
	"github.com/zeebo/errs"
)

type opType int

const (
	opPut    opType = 1
	opDelete opType = 2
)

type stagedEntry struct {
	op       opType
	path     string
	metadata *manifest.Metadata
	source   io.ReadCloser
	stream   *manifest.Stream
}

type Session struct {
	backend   backends.Backend
	staging   []*stagedEntry
	unflushed []*stagedEntry
}

func newSession(backend backends.Backend) *Session {
	s := &Session{
		backend: backend,
	}
	return s
}

func (s *Session) List(ctx context.Context, prefix string, cb func(context.Context, manifest.Entry) error) error {
	panic("TODO")
}

func (s *Session) Open(ctx context.Context, path string) (*manifest.Metadata, *stream.Stream, error) {
	panic("TODO")
}

func (s *Session) Delete(ctx context.Context, path string) error {
	s.staging = append(s.staging, &stagedEntry{
		op:   opDelete,
		path: path,
	})
	return nil
}

// PutFile causes the Session to take ownership of the data io.ReadCloser and will close it when the Session
// either uses the data or closes itself.
func (s *Session) PutFile(ctx context.Context, path string, creation, modified time.Time, mode uint32, data io.ReadCloser) error {
	creationPB, modifiedPB, err := convertTime(creation, modified)
	if err != nil {
		return errs.Combine(err, data.Close())
	}

	entry := &stagedEntry{
		op:   opPut,
		path: path,
		metadata: &manifest.Metadata{
			Type:     manifest.Metadata_FILE,
			Creation: creationPB,
			Modified: modifiedPB,
			Mode:     mode,
		},
		source: data,
	}

	s.staging = append(s.staging, entry)
	s.unflushed = append(s.unflushed, entry)

	return nil
}

func (s *Session) PutDir(ctx context.Context, path string, creation, modified time.Time, mode uint32) error {
	creationPB, modifiedPB, err := convertTime(creation, modified)
	if err != nil {
		return err
	}

	s.staging = append(s.staging, &stagedEntry{
		op:   opPut,
		path: path,
		metadata: &manifest.Metadata{
			Type:     manifest.Metadata_DIR,
			Creation: creationPB,
			Modified: modifiedPB,
			Mode:     mode,
		},
	})

	return nil
}

func (s *Session) PutSymlink(ctx context.Context, path string, creation, modified time.Time, mode uint32, target string) error {
	creationPB, modifiedPB, err := convertTime(creation, modified)
	if err != nil {
		return err
	}

	s.staging = append(s.staging, &stagedEntry{
		op:   opPut,
		path: path,
		metadata: &manifest.Metadata{
			Type:       manifest.Metadata_SYMLINK,
			Creation:   creationPB,
			Modified:   modifiedPB,
			Mode:       mode,
			LinkTarget: target,
		}})

	return nil
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

func (s *Session) Flush(ctx context.Context) error {
	panic("TODO")
}

func (s *Session) Commit(ctx context.Context) error {
	err := s.Flush(ctx)
	if err != nil {
		return err
	}
	panic("TODO")
}

func (s *Session) Close(ctx context.Context) error {
	unflushed := s.unflushed
	s.staging, s.unflushed = nil, nil
	var group errs.Group
	for _, entry := range unflushed {
		group.Add(entry.source.Close())
	}
	return group.Err()
}
