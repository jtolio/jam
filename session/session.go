package session

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/zeebo/errs"

	"github.com/jtolds/jam/backends"
	"github.com/jtolds/jam/blobs"
	"github.com/jtolds/jam/hashdb"
	"github.com/jtolds/jam/manifest"
	"github.com/jtolds/jam/pathdb"
)

type ReadSeekCloser interface {
	io.Reader
	io.Seeker
	io.Closer
}

type Session struct {
	backend backends.Backend
	paths   *pathdb.DB
	blobs   *blobs.Store
	hashes  *hashdb.DB
}

func newSession(backend backends.Backend, paths *pathdb.DB, blobStore *blobs.Store, hashes *hashdb.DB) *Session {
	return &Session{
		backend: backend,
		paths:   paths,
		blobs:   blobStore,
		hashes:  hashes,
	}
}

func (s *Session) Delete(ctx context.Context, path string) error {
	return s.paths.Delete(ctx, path)
}

func (s *Session) DeleteAll(ctx context.Context, re *regexp.Regexp) error {
	return s.paths.DeleteAll(ctx, re)
}

// PutFile causes the Session to take ownership of the data io.ReadCloser and will close it when the Session
// either uses the data or closes itself.
func (s *Session) PutFile(ctx context.Context, path string, creation, modified time.Time, mode uint32, data ReadSeekCloser) error {
	if strings.HasSuffix(path, "/") {
		return fmt.Errorf("file paths cannot end with a '/': %q", path)
	}
	creationPB, modifiedPB, err := convertTime(creation, modified)
	if err != nil {
		return errs.Combine(err, data.Close())
	}

	startOffset, err := data.Seek(0, io.SeekCurrent)
	if err != nil {
		return errs.Combine(err, data.Close())
	}

	hasher := sha256.New()
	size, err := io.Copy(hasher, data)
	if err != nil {
		return errs.Combine(err, data.Close())
	}

	var hashAlloc [sha256.Size]byte
	hash := hasher.Sum(hashAlloc[:])

	_, err = data.Seek(startOffset, io.SeekStart)
	if err != nil {
		return errs.Combine(err, data.Close())
	}

	content := &manifest.Content{
		Metadata: &manifest.Metadata{
			Type:     manifest.Metadata_FILE,
			Creation: creationPB,
			Modified: modifiedPB,
			Mode:     mode,
		},
		Hash: hash,
	}

	exists, err := s.hashes.Has(ctx, string(hash))
	if err != nil {
		return errs.Combine(err, data.Close())
	}

	if exists {
		err = data.Close()
		if err != nil {
			return err
		}
	} else {
		// Put closes data
		err = s.blobs.Put(ctx, data, size, func(ctx context.Context, stream *manifest.Stream) error {
			return s.hashes.Put(ctx, string(hash), stream)
		})
		if err != nil {
			return err
		}
	}

	return s.paths.Put(ctx, path, content)
}

func (s *Session) PutSymlink(ctx context.Context, path string, creation, modified time.Time, mode uint32, target string) error {
	if strings.HasSuffix(path, "/") {
		return fmt.Errorf("file paths cannot end with a '/': %q", path)
	}
	creationPB, modifiedPB, err := convertTime(creation, modified)
	if err != nil {
		return err
	}

	content := &manifest.Content{
		Metadata: &manifest.Metadata{
			Type:       manifest.Metadata_SYMLINK,
			Creation:   creationPB,
			Modified:   modifiedPB,
			Mode:       mode,
			LinkTarget: target,
		},
	}

	return s.paths.Put(ctx, path, content)
}

// Rename renames paths using regexp.ReplaceAllString (replacement can have
// regexp expansions). See the docs for regexp.ReplaceAllString
func (s *Session) Rename(ctx context.Context, re *regexp.Regexp, replacement string) error {
	return s.paths.Rename(ctx, re, replacement)
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
	err := s.blobs.Flush(ctx)
	if err != nil {
		return err
	}
	return s.hashes.Flush(ctx)
}

func (s *Session) Commit(ctx context.Context) (err error) {
	err = s.Flush(ctx)
	if err != nil {
		return err
	}
	rc, err := s.paths.Serialize(ctx)
	if err != nil {
		return err
	}
	defer func() {
		err = errs.Combine(err, rc.Close())
	}()
	// TODO: make sure this timestamp is strictly newer than all previous
	// timestamps, and make sure you can't delete the newest timestamp,
	// to avoid key reuse with different snapshots with the same timestamp
	return s.backend.Put(ctx, timestampToPath(time.Now()), rc)
}

func (s *Session) Close() error {
	return s.paths.Close()
}
