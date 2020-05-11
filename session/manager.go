package session

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/zeebo/errs"

	"github.com/jtolds/jam/backends"
	"github.com/jtolds/jam/blobs"
	"github.com/jtolds/jam/hashdb"
	"github.com/jtolds/jam/pathdb"
	"github.com/jtolds/jam/utils"
)

const (
	pathPrefix = "manifests/"
	timeFormat = "2006/01/02/15-04-05.000000000"
)

func timestampToPath(timestamp time.Time) string {
	return pathPrefix + timestamp.UTC().Format(timeFormat)
}

func pathToTimestamp(path string) (time.Time, error) {
	if !strings.HasPrefix(path, pathPrefix) {
		return time.Time{}, errs.New("backend had unexpected behavior: path returned does not start with %q: %q", pathPrefix, path)
	}
	ts, err := time.ParseInLocation(timeFormat, strings.TrimPrefix(path, pathPrefix), time.UTC)
	return ts, errs.Wrap(err)
}

type Manager struct {
	backend backends.Backend
	blobs   *blobs.Store
	hashes  *hashdb.DB
}

func NewManager(backend backends.Backend, blobStore *blobs.Store, hashes *hashdb.DB) *Manager {
	return &Manager{
		backend: backend,
		blobs:   blobStore,
		hashes:  hashes,
	}
}

func (s *Manager) ListSnapshots(ctx context.Context,
	cb func(ctx context.Context, timestamp time.Time) error) error {
	// TODO: backend.List is not ordered. we could use the fact that manifests are stored
	//		using timeFormat format and list by years and months in decreasing order to get
	// 		an order
	return errs.Wrap(s.backend.List(ctx, pathPrefix, func(ctx context.Context, path string) error {
		timestamp, err := pathToTimestamp(path)
		if err != nil {
			utils.L(ctx).Urgentf("invalid manifest format: %q, skipping. error: %v", path, err)
			return nil
		}
		return cb(ctx, timestamp)
	}))
}

func (s *Manager) latestTimestamp(ctx context.Context) (time.Time, error) {
	var latest time.Time
	// TODO: there is probably a better way here
	err := s.ListSnapshots(ctx, func(ctx context.Context, timestamp time.Time) error {
		if latest.IsZero() || timestamp.After(latest) {
			latest = timestamp
		}
		return nil
	})
	if err != nil {
		return latest, err
	}
	return latest, nil
}

func (s *Manager) LatestSnapshot(ctx context.Context) (*Snapshot, time.Time, error) {
	latest, err := s.latestTimestamp(ctx)
	if err != nil {
		return nil, time.Time{}, err
	}
	if latest.IsZero() {
		return nil, time.Time{}, fmt.Errorf("no snapshots exist yet")
	}
	snap, err := s.OpenSnapshot(ctx, latest)
	return snap, latest, err
}

func (s *Manager) openPathDB(ctx context.Context, timestamp time.Time) (*pathdb.DB, error) {
	err := s.confirmSnapExists(ctx, timestamp)
	if err != nil {
		return nil, err
	}
	rc, err := s.backend.Get(ctx, timestampToPath(timestamp), 0, -1)
	if err != nil {
		return nil, err
	}
	defer func() {
		err = errs.Combine(err, rc.Close())
	}()

	return pathdb.Open(ctx, s.backend, s.blobs, rc)
}

func (s *Manager) OpenSnapshot(ctx context.Context, timestamp time.Time) (*Snapshot, error) {
	db, err := s.openPathDB(ctx, timestamp)
	if err != nil {
		return nil, err
	}
	return newSnapshot(s.backend, db, s.blobs, s.hashes), nil
}

func (s *Manager) NewSession(ctx context.Context) (*Session, error) {
	latest, err := s.latestTimestamp(ctx)
	if err != nil {
		return nil, err
	}
	var db *pathdb.DB
	if latest.IsZero() {
		db = pathdb.New(s.backend, s.blobs)
	} else {
		db, err = s.openPathDB(ctx, latest)
		if err != nil {
			return nil, err
		}
	}
	return newSession(s.backend, db, s.blobs, s.hashes), nil
}

func (s *Manager) RevertTo(ctx context.Context, timestamp time.Time) (*Session, error) {
	db, err := s.openPathDB(ctx, timestamp)
	if err != nil {
		return nil, err
	}
	return newSession(s.backend, db, s.blobs, s.hashes), nil
}

func (s *Manager) DeleteSnapshot(ctx context.Context, timestamp time.Time) error {
	latest, err := s.latestTimestamp(ctx)
	if err != nil {
		return err
	}
	if latest.IsZero() {
		return fmt.Errorf("no snapshots exist yet")
	}
	if !latest.After(timestamp) {
		return fmt.Errorf("can't remove latest snapshot")
	}
	err = s.confirmSnapExists(ctx, timestamp)
	if err != nil {
		return err
	}
	return s.backend.Delete(ctx, timestampToPath(timestamp))
}

var escapeList = fmt.Errorf("escaping list")
var noSnap = fmt.Errorf("snap does not exist")

func (s *Manager) confirmSnapExists(ctx context.Context, timestamp time.Time) error {
	found := false
	timestampPath := timestampToPath(timestamp)
	err := s.backend.List(ctx, filepath.Dir(timestampPath)+"/", func(ctx context.Context, path string) error {
		if path == timestampPath {
			found = true
			return escapeList
		}
		return nil
	})
	if err != nil && !errors.Is(err, escapeList) {
		return err
	}
	if !found {
		return noSnap
	}
	return nil
}
