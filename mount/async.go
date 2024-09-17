package mount

import (
	"context"
	"sync"

	"github.com/zeebo/errs"

	"github.com/jtolio/jam/manifest"
	"github.com/jtolio/jam/session"
	"github.com/jtolio/jam/streams"
)

type asyncFSSnap struct {
	constructor func(context.Context) (FSSnap, error)
	mtx         sync.Mutex
	snap        FSSnap
	initerr     error
}

func AsyncFSSnap(ctx context.Context, constructor func(context.Context) (FSSnap, error)) FSSnap {
	a := &asyncFSSnap{constructor: constructor}
	go a.init(ctx)
	return a
}

func (a *asyncFSSnap) init(ctx context.Context) (FSSnap, error) {
	a.mtx.Lock()
	defer a.mtx.Unlock()
	if a.initerr != nil {
		return nil, a.initerr
	}
	if a.snap == nil {
		constructor := a.constructor
		a.constructor = nil
		if constructor == nil {
			return nil, errs.New("already constructed")
		}
		snap, err := constructor(ctx)
		if err != nil {
			a.initerr = err
			return nil, err
		}
		a.snap = snap
	}
	return a.snap, nil
}

func (a *asyncFSSnap) Open(ctx context.Context, path string) (*manifest.Metadata, *streams.Stream, error) {
	snap, err := a.init(ctx)
	if err != nil {
		return nil, nil, err
	}
	return snap.Open(ctx, path)
}

func (a *asyncFSSnap) HasPrefix(ctx context.Context, prefix string) (bool, error) {
	snap, err := a.init(ctx)
	if err != nil {
		return false, err
	}
	return snap.HasPrefix(ctx, prefix)
}

func (a *asyncFSSnap) List(ctx context.Context, prefix string, recursive bool,
	cb func(ctx context.Context, entry *session.ListEntry) error) error {
	snap, err := a.init(ctx)
	if err != nil {
		return err
	}
	return snap.List(ctx, prefix, recursive, cb)
}
