package session

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"io"
	"time"

	"github.com/jtolds/jam/backends"
	"github.com/jtolds/jam/pkg/manifest"
	"github.com/jtolds/jam/pkg/stream"
)

func idGen() string {
	var buf [32]byte
	_, err := rand.Read(buf[:])
	if err != nil {
		panic(err)
	}
	return base64.URLEncoding.EncodeToString(string(buf[:])), nil
}

type Session struct {
	backend backends.Backend
	current *Concatenator
}

func newSession(backend backends.Backend) *Session {
	s := &Session{
		backend: backend,
	}
	return s
}

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

func (s *Session) WriteFile(ctx context.Context, path string, creation, modified time.Time, mode uint32, data io.Reader) error {
	panic("TODO")
}

func (s *Session) WriteDir(ctx context.Context, path string, creation, modified time.Time, mode uint32) error {
	panic("TODO")
}

func (s *Session) WriteSymlink(ctx context.Context, path string, creation, modified time.Time, mode uint32, target string) error {
	panic("TODO")
}

func (s *Session) Commit(ctx context.Context) error {
	panic("TODO")
}

func (s *Session) List(ctx context.Context, prefix string, cb func(context.Context, manifest.Entry) error) error {
	panic("TODO")
}

func (s *Session) OpenFile(ctx context.Context, path string) (*manifest.Metadata, *stream.Stream, error) {
	panic("TODO")
}

func (s *Session) Close() error {
	panic("TODO")
}

type SessionManager struct {
	backend backends.Backend
}

func NewSessionManager(backend backends.Backend) *SessionManager {
	return &SessionManager{backend: backend}
}

func (s *SessionManager) ListSessions(ctx context.Context, cb func(context.Context, time.Time) error) error {
	panic("TODO")
}

func (s *SessionManager) LatestSession(ctx context.Context) (*Session, error) {
	panic("TODO")
}

func (s *SessionManager) OpenSession(ctx context.Context, timestamp time.Time) (*Session, error) {
	panic("TODO")
}
