package hashdb

import (
	"context"
	"sync"

	"github.com/zeebo/errs"

	"github.com/jtolio/jam/manifest"
)

type asyncHashDB struct {
	constructor func(context.Context) (DB, error)
	mtx         sync.Mutex
	hashDB      DB
	initerr     error
}

func AsyncHashDB(ctx context.Context, constructor func(context.Context) (DB, error)) DB {
	db := &asyncHashDB{constructor: constructor}
	go db.init(ctx)
	return db
}

func (a *asyncHashDB) Close() error {
	a.mtx.Lock()
	hashDB := a.hashDB
	a.hashDB = nil
	a.constructor = nil
	initerr := a.initerr
	a.mtx.Unlock()
	if hashDB != nil {
		return errs.Combine(hashDB.Close(), initerr)
	}
	return initerr
}

func (a *asyncHashDB) init(ctx context.Context) (DB, error) {
	a.mtx.Lock()
	defer a.mtx.Unlock()
	if a.initerr != nil {
		return nil, a.initerr
	}
	if a.hashDB == nil {
		constructor := a.constructor
		a.constructor = nil
		if constructor == nil {
			return nil, errs.New("already constructed")
		}
		hashDB, err := constructor(ctx)
		if err != nil {
			a.initerr = err
			return nil, err
		}
		a.hashDB = hashDB
	}
	return a.hashDB, nil
}

func (a *asyncHashDB) Coalesce(ctx context.Context) error {
	hashDB, err := a.init(ctx)
	if err != nil {
		return err
	}
	return hashDB.Coalesce(ctx)
}

func (a *asyncHashDB) Flush(ctx context.Context) error {
	hashDB, err := a.init(ctx)
	if err != nil {
		return err
	}
	return hashDB.Flush(ctx)
}

func (a *asyncHashDB) Has(ctx context.Context, hash string) (exists bool, err error) {
	hashDB, err := a.init(ctx)
	if err != nil {
		return false, err
	}
	return hashDB.Has(ctx, hash)
}

func (a *asyncHashDB) Iterate(ctx context.Context,
	cb func(ctx context.Context, hash, hashset string, data *manifest.Stream) error) error {
	hashDB, err := a.init(ctx)
	if err != nil {
		return err
	}
	return hashDB.Iterate(ctx, cb)
}

func (a *asyncHashDB) Lookup(ctx context.Context, hash string) (*manifest.Stream, error) {
	hashDB, err := a.init(ctx)
	if err != nil {
		return nil, err
	}
	return hashDB.Lookup(ctx, hash)
}

func (a *asyncHashDB) Put(ctx context.Context, hash string, data *manifest.Stream) error {
	hashDB, err := a.init(ctx)
	if err != nil {
		return err
	}
	return hashDB.Put(ctx, hash, data)
}

func (a *asyncHashDB) Split(ctx context.Context) error {
	hashDB, err := a.init(ctx)
	if err != nil {
		return err
	}
	return hashDB.Split(ctx)
}
