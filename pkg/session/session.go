package session

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/zeebo/errs"

	"github.com/jtolds/jam/backends"
	"github.com/jtolds/jam/pkg/blobs"
	"github.com/jtolds/jam/pkg/manifest"
	"github.com/jtolds/jam/pkg/pathdb"
	"github.com/jtolds/jam/pkg/streams"
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
	stream   *manifest.Stream
}

type Session struct {
	backend backends.Backend
	paths   *pathdb.DB
	blobs   *blobs.Store
	staging []*stagedEntry
}

func newSession(backend backends.Backend, paths *pathdb.DB, blobStore *blobs.Store) *Session {
	s := &Session{
		backend: backend,
		paths:   paths,
		blobs:   blobStore,
	}
	return s
}

func (s *Session) List(ctx context.Context, prefix string, recursive bool,
	cb func(ctx context.Context, path string, meta *manifest.Metadata, data *streams.Stream) error) error {
	// TODO: it's weird that we're passing an open stream here. we need to make it way clearer
	// how to deal with the stream life cycle somehow.
	return s.paths.List(ctx, prefix, recursive,
		func(ctx context.Context, path string, content *manifest.Content) error {
			stream, err := streams.Open(ctx, s.backend, content.Data)
			if err != nil {
				return err
			}
			return cb(ctx, path, content.Metadata, stream)
		})
}

var ErrNotFound = fmt.Errorf("file not found")

func (s *Session) Open(ctx context.Context, path string) (*manifest.Metadata, *streams.Stream, error) {
	content, err := s.paths.Get(ctx, path)
	if err != nil {
		return nil, nil, err
	}
	if content == nil {
		return nil, nil, ErrNotFound
	}
	stream, err := streams.Open(ctx, s.backend, content.Data)
	return content.Metadata, stream, err
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
	if strings.HasSuffix(path, "/") {
		return fmt.Errorf("file paths cannot end with a '/': %q", path)
	}
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
	}

	s.staging = append(s.staging, entry)

	return s.blobs.Put(ctx, data, func(stream *manifest.Stream) {
		entry.stream = stream
	})
}

func (s *Session) PutSymlink(ctx context.Context, path string, creation, modified time.Time, mode uint32, target string) error {
	if strings.HasSuffix(path, "/") {
		return fmt.Errorf("file paths cannot end with a '/': %q", path)
	}
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
	return s.blobs.Flush(ctx)
}

func (s *Session) Commit(ctx context.Context) (err error) {
	if len(s.staging) == 0 {
		return nil
	}
	err = s.Flush(ctx)
	if err != nil {
		return err
	}
	for _, entry := range s.staging {
		switch entry.op {
		case opPut:
			err = s.paths.Put(ctx, entry.path, &manifest.Content{
				Metadata: entry.metadata,
				Data:     entry.stream,
			})
			if err != nil {
				return err
			}
		case opDelete:
			err = s.paths.Delete(ctx, entry.path)
			if err != nil {
				return err
			}
		default:
			panic(fmt.Sprintf("unknown op type: %q", entry.op))
		}
	}

	rc, err := s.paths.Serialize(ctx)
	if err != nil {
		return err
	}
	defer func() {
		err = errs.Combine(err, rc.Close())
	}()
	return s.backend.Put(ctx, timestampToPath(time.Now()), rc)
}

func (s *Session) Close() error {
	s.staging = nil
	return s.paths.Close()
}