package session

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/zeebo/errs"

	"github.com/jtolio/jam/backends"
	"github.com/jtolio/jam/blobs"
	"github.com/jtolio/jam/hashdb"
	"github.com/jtolio/jam/manifest"
	"github.com/jtolio/jam/pathdb"
	"github.com/jtolio/jam/utils"
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
	hashes  hashdb.DB
	pending map[string]bool
}

func newSession(backend backends.Backend, paths *pathdb.DB, blobStore *blobs.Store, hashes hashdb.DB) *Session {
	return &Session{
		backend: backend,
		paths:   paths,
		blobs:   blobStore,
		hashes:  hashes,
		pending: map[string]bool{},
	}
}

func (s *Session) Delete(ctx context.Context, path string) (removed bool, err error) {
	return s.paths.Delete(ctx, path)
}

func (s *Session) DeleteAll(ctx context.Context, matcher func(string) bool) (removed int, err error) {
	return s.paths.DeleteAll(ctx, matcher)
}

// PutFile causes the Session to take ownership of the data io.ReadCloser and will close it when the Session
// either uses the data or closes itself.
func (s *Session) PutFile(ctx context.Context, path string, creation, modified time.Time, mode uint32,
	data ReadSeekCloser) (state pathdb.PutState, err error) {
	if strings.HasSuffix(path, "/") {
		return pathdb.PutStateUnchanged, fmt.Errorf("file paths cannot end with a '/': %q", path)
	}
	creationPB, modifiedPB, err := convertTime(creation, modified)
	if err != nil {
		return pathdb.PutStateUnchanged, errs.Combine(err, data.Close())
	}

	startOffset, err := data.Seek(0, io.SeekCurrent)
	if err != nil {
		return pathdb.PutStateUnchanged, errs.Combine(err, data.Close())
	}

	hasher := sha256.New()
	size, err := io.Copy(hasher, data)
	if err != nil {
		return pathdb.PutStateUnchanged, errs.Combine(err, data.Close())
	}

	var hashAlloc [sha256.Size]byte
	hash := hasher.Sum(hashAlloc[:0])

	_, err = data.Seek(startOffset, io.SeekStart)
	if err != nil {
		return pathdb.PutStateUnchanged, errs.Combine(err, data.Close())
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

	hashStr := string(hash)

	exists, err := s.hashes.Has(ctx, hashStr)
	if err != nil {
		return pathdb.PutStateUnchanged, errs.Combine(err, data.Close())
	}

	if exists || s.pending[hashStr] {
		utils.L(ctx).Debugf("data for %q is duplicate", path)
		err = data.Close()
		if err != nil {
			return pathdb.PutStateUnchanged, err
		}

	} else {
		s.pending[hashStr] = true

		// Put closes data so we don't have to call Close
		err = s.blobs.Put(ctx,
			newHashConfirmReader(data, sha256.New(), hash, fmt.Sprintf("file changed while reading: %q", path)),
			size, &sortKey{col1: filepath.Dir(path), col2: size},
			func(ctx context.Context, stream *manifest.Stream, lastOfBlob bool) error {
				utils.L(ctx).Normalf("stored data for %q", path)
				err := s.hashes.Put(ctx, string(hash), stream)
				if err != nil {
					return err
				}
				delete(s.pending, hashStr)
				if lastOfBlob {
					return s.hashes.Flush(ctx)
				}
				return nil
			})
		if err != nil {
			return pathdb.PutStateUnchanged, err
		}
	}

	return s.paths.Put(ctx, path, content)
}

func (s *Session) PutSymlink(ctx context.Context, path string, creation, modified time.Time, mode uint32, target string) (
	state pathdb.PutState, err error) {
	if strings.HasSuffix(path, "/") {
		return pathdb.PutStateUnchanged, fmt.Errorf("file paths cannot end with a '/': %q", path)
	}
	creationPB, modifiedPB, err := convertTime(creation, modified)
	if err != nil {
		return pathdb.PutStateUnchanged, err
	}

	content := &manifest.Content{
		Metadata: &manifest.Metadata{
			Type:       manifest.Metadata_SYMLINK,
			Creation:   creationPB,
			Modified:   modifiedPB,
			Mode:       mode,
			LinkTarget: []byte(target),
		},
	}

	utils.L(ctx).Debugf("stored symlink %q", path)

	return s.paths.Put(ctx, path, content)
}

// Rename renames paths using regexp.ReplaceAllString (replacement can have
// regexp expansions). See the docs for regexp.ReplaceAllString
func (s *Session) Rename(ctx context.Context, re *regexp.Regexp, replacement string) (renamed int, err error) {
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
	if !s.paths.Changed() {
		utils.L(ctx).Normalf("no changes detected, skipping new manifest")
		return nil
	}
	// TODO: make sure this timestamp is strictly newer than all previous
	// timestamps, and make sure you can't delete the newest timestamp,
	// to avoid key reuse with different snapshots with the same timestamp
	ts := time.Now()
	err = s.paths.SerializeTo(ctx, timestampToPath(ts))
	if err != nil {
		return err
	}

	utils.L(ctx).Normalf("wrote manifest for %v", ts)
	return nil
}

func (s *Session) Close() error {
	return s.paths.Close()
}

type sortKey struct {
	col1 string
	col2 int64
}

func (a *sortKey) Less(bi blobs.SortKey) bool {
	b := bi.(*sortKey)
	if a.col1 < b.col1 {
		return true
	}
	if a.col1 > b.col1 {
		return false
	}
	return a.col2 < b.col2
}

func (s *Session) List(ctx context.Context, prefix string, recursive bool,
	cb func(ctx context.Context, path string, prefix bool) error) error {
	return s.paths.List(ctx, prefix, recursive,
		func(ctx context.Context, path string, content *manifest.Content) error {
			return cb(ctx, path, content == nil)
		})
}
