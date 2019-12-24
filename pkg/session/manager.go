package session

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jtolds/jam/backends"
	"github.com/jtolds/jam/pkg/manifest"
	"github.com/jtolds/jam/pkg/pathdb"
	"github.com/jtolds/jam/pkg/stream"
)

type Logger interface {
	Printf(format string, v ...interface{})
}

type Snapshot interface {
	List(ctx context.Context, prefix string, cb func(context.Context, manifest.Entry) error) error
	Open(ctx context.Context, path string) (*manifest.Metadata, *stream.Stream, error)
}

type SessionManager struct {
	backend      backends.Backend
	logger       Logger
	blobSize     int64
	maxUnflushed int
}

func NewSessionManager(backend backends.Backend, logger Logger, blobSize int64, maxUnflushed int) *SessionManager {
	if logger == nil {
		logger = log.New(os.Stderr, "", log.LstdFlags)
	}
	return &SessionManager{
		backend:      backend,
		logger:       logger,
		blobSize:     blobSize,
		maxUnflushed: maxUnflushed,
	}
}

func (s *SessionManager) ListSnapshots(ctx context.Context,
	cb func(ctx context.Context, timestamp time.Time) error) error {
	// TODO: backend.List is not ordered. it might be nice if we added structure (named the files
	//		after years/months/days with folders for each) so we could return these ordered.
	return s.backend.List(ctx, "meta/", func(ctx context.Context, path string) error {
		if !strings.HasPrefix("meta/", path) {
			return fmt.Errorf("backend had unexpected behavior: path returned does not start with 'meta/': %q", path)
		}
		nsecs, err := strconv.ParseInt(path[5:], 10, 64)
		if err != nil {
			s.logger.Printf("found invalid manifest timestamp: %q, skipping", path)
			return nil
		}
		return cb(ctx, time.Unix(0, nsecs))
	})
}

func (s *SessionManager) latestTimestamp(ctx context.Context) (time.Time, error) {
	var latest time.Time
	err := s.ListSnapshots(ctx, func(ctx context.Context, timestamp time.Time) error {
		if latest.IsZero() || timestamp.After(latest) {
			latest = timestamp
		}
		return nil
	})
	if err != nil {
		return latest, err
	}
	if latest.IsZero() {
		return latest, fmt.Errorf("no snapshots exist yet")
	}
	return latest, nil
}

func (s *SessionManager) LatestSnapshot(ctx context.Context) (Snapshot, error) {
	latest, err := s.latestTimestamp(ctx)
	if err != nil {
		return nil, err
	}
	return s.OpenSnapshot(ctx, latest)
}

func (s *SessionManager) openSession(ctx context.Context, timestamp time.Time) (*Session, error) {
	db, err := pathdb.Open(ctx, s.backend, "meta/"+fmt.Sprint(timestamp.UnixNano()))
	if err != nil {
		return nil, err
	}
	return newSession(s.backend, db, s.blobSize, s.maxUnflushed), nil
}

func (s *SessionManager) OpenSnapshot(ctx context.Context, timestamp time.Time) (Snapshot, error) {
	return s.openSession(ctx, timestamp)
}

func (s *SessionManager) NewSession(ctx context.Context) (*Session, error) {
	latest, err := s.latestTimestamp(ctx)
	if err != nil {
		return nil, err
	}
	return s.openSession(ctx, latest)
}
