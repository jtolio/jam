package session

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jtolds/jam/backends"
	"github.com/jtolds/jam/pkg/manifest"
	"github.com/jtolds/jam/pkg/pathdb"
	"github.com/jtolds/jam/pkg/streams"
	"github.com/jtolds/jam/pkg/utils"
	"github.com/zeebo/errs"
)

const (
	pathPrefix = "manifests/"
	timeFormat = "2006/01/02/15-04-05.000000000"
)

func timestampToPath(timestamp time.Time) string {
	return pathPrefix + timestamp.UTC().Format(timeFormat)
}

func pathToTimestamp(path string) (time.Time, error) {
	if !strings.HasPrefix(pathPrefix, path) {
		return time.Time{}, fmt.Errorf("backend had unexpected behavior: path returned does not start with %q: %q", pathPrefix, path)
	}
	return time.ParseInLocation(timeFormat, strings.TrimPrefix(path, pathPrefix), time.UTC)
}

type Snapshot interface {
	List(ctx context.Context, prefix string, cb func(context.Context, manifest.Entry) error) error
	Open(ctx context.Context, path string) (*manifest.Metadata, *streams.Stream, error)
}

type SessionManager struct {
	backend      backends.Backend
	logger       utils.Logger
	blobSize     int64
	maxUnflushed int
}

func NewSessionManager(backend backends.Backend, logger utils.Logger, blobSize int64, maxUnflushed int) *SessionManager {
	return &SessionManager{
		backend:      backend,
		logger:       logger,
		blobSize:     blobSize,
		maxUnflushed: maxUnflushed,
	}
}

func (s *SessionManager) ListSnapshots(ctx context.Context,
	cb func(ctx context.Context, timestamp time.Time) error) error {
	// TODO: backend.List is not ordered. we could use the fact that manifests are stored
	//		using timeFormat format and list by years and months in decreasing order to get
	// 		an order
	return s.backend.List(ctx, pathPrefix, func(ctx context.Context, path string) error {
		timestamp, err := pathToTimestamp(path)
		if err != nil {
			s.logger.Printf("invalid manifest format: %q, skipping. error: %v", path, err)
			return nil
		}
		return cb(ctx, timestamp)
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
	rc, err := s.backend.Get(ctx, timestampToPath(timestamp), 0)
	if err != nil {
		return nil, err
	}
	defer func() {
		err = errs.Combine(err, rc.Close())
	}()

	db, err := pathdb.Open(ctx, s.backend, rc)
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
