package session

import (
	"context"
	"errors"
	"fmt"
	"sort"
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
	ManifestPrefix = "manifests/"
	timeFormat     = "2006/01/02/15-04-05.000000000"
)

func timestampToPath(timestamp time.Time) string {
	return ManifestPrefix + timestamp.UTC().Format(timeFormat)
}

func pathToTimestamp(path string) (time.Time, error) {
	if !strings.HasPrefix(path, ManifestPrefix) {
		return time.Time{}, errs.New(
			"backend had unexpected behavior: path returned does not start with %q: %q",
			ManifestPrefix, path)
	}
	ts, err := time.ParseInLocation(timeFormat,
		strings.TrimPrefix(path, ManifestPrefix), time.UTC)
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

// ListSnapshots returns snapshots newest to oldest
func (s *Manager) ListSnapshots(ctx context.Context,
	cb func(ctx context.Context, timestamp time.Time) error) error {
	// TODO: backend.List is not ordered. we could use the fact that manifests are stored
	//		using timeFormat format and list by years and months in decreasing order to get
	// 		an order
	var timestamps []time.Time
	err := s.backend.List(ctx, ManifestPrefix,
		func(ctx context.Context, path string) error {
			timestamp, err := pathToTimestamp(path)
			if err != nil {
				utils.L(ctx).Urgentf("invalid manifest format: %q, skipping. error: %v", path, err)
				return nil
			}
			timestamps = append(timestamps, timestamp)
			return nil
		})
	if err != nil {
		return errs.Wrap(err)
	}
	sort.Slice(timestamps, func(i, j int) bool {
		return timestamps[i].After(timestamps[j])
	})
	for _, timestamp := range timestamps {
		err = cb(ctx, timestamp)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Manager) latestTimestamp(ctx context.Context) (time.Time, error) {
	var latest time.Time
	exitEarly := errors.New("list-snapshot-early-exit")
	err := s.ListSnapshots(ctx, func(ctx context.Context, timestamp time.Time) error {
		latest = timestamp
		return exitEarly
	})
	if err != nil {
		if errors.Is(err, exitEarly) {
			return latest, nil
		}
		return latest, err
	}
	if latest.IsZero() {
		return latest, nil
	}
	return latest, errs.New("unexpected codepath!")
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
	return s.backend.Delete(ctx, timestampToPath(timestamp))
}
