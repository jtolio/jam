package session

import (
	"context"
	"time"

	"github.com/jtolds/jam/backends"
)

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

func (s *SessionManager) NewSession(ctx context.Context) (*Mutation, error) {
	panic("TODO")
}
