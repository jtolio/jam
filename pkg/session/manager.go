package session

import (
	"context"
	"time"

	"github.com/jtolds/jam/backends"
	"github.com/jtolds/jam/pkg/manifest"
	"github.com/jtolds/jam/pkg/stream"
)

type SessionManager struct {
	backend backends.Backend
}

func NewSessionManager(backend backends.Backend) *SessionManager {
	return &SessionManager{backend: backend}
}

func (s *SessionManager) ListSnapshots(ctx context.Context, cb func(context.Context, time.Time) error) error {
	panic("TODO")
}

func (s *SessionManager) LatestSnapshot(ctx context.Context) (Snapshot, error) {
	panic("TODO")
}

func (s *SessionManager) OpenSnapshot(ctx context.Context, timestamp time.Time) (Snapshot, error) {
	panic("TODO")
}

func (s *SessionManager) NewSession(ctx context.Context) (*Session, error) {
	panic("TODO")
}

type Snapshot interface {
	List(ctx context.Context, prefix string, cb func(context.Context, manifest.Entry) error) error
	Open(ctx context.Context, path string) (*manifest.Metadata, *stream.Stream, error)
}
