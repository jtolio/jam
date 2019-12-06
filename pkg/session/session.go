package session

import (
	"context"
	"io"
	"time"

	"github.com/jtolds/jam/backends"
	"github.com/jtolds/jam/pkg/manifest"
)

type File struct{}

func (f *File) Read(p []byte) (n int, err error) {
	panic("TODO")
}

func (f *File) Close() error {
	panic("TODO")
}

func (f *File) Seek(offset int64, whence int) (int64, error) {
	panic("TODO")
}

type Session struct{}

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

func (s *Session) OpenFile(ctx context.Context, path string) (*File, error) {
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
