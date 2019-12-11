package session

import (
	"context"

	"github.com/jtolds/jam/backends"
	"github.com/jtolds/jam/pkg/manifest"
	"github.com/jtolds/jam/pkg/stream"
)

type Session struct {
	backend backends.Backend
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
