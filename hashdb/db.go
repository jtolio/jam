package hashdb

import (
	"bytes"
	"compress/zlib"
	"context"
	"crypto/sha256"
	"io"

	"github.com/zeebo/errs"

	"github.com/jtolio/jam/backends"
	"github.com/jtolio/jam/manifest"
	"github.com/jtolio/jam/streams"
	"github.com/jtolio/jam/utils"
)

const versionHeader = "jam-v0\n"
const HashPrefix = "hash/"
const SmallHashsetSuffix = ".hs"
const SmallHashsetThreshold = 64 * 1024

type DB interface {
	Close() error
	Coalesce(context.Context) error
	Flush(context.Context) error
	Has(ctx context.Context, hash string) (exists bool, err error)
	Iterate(context.Context,
		func(ctx context.Context, hash, hashset string, data *manifest.Stream) error) error
	Lookup(ctx context.Context, hash string) (*manifest.Stream, error)
	Put(ctx context.Context, hash string, data *manifest.Stream) error
	Split(context.Context) error
}

type dbImpl struct {
	backend backends.Backend

	// TODO: do an LSM tree instead of putting all of this in RAM
	existing map[string]*manifest.Stream
	new      map[string]*manifest.Stream
	source   map[string]string
	paths    []string
}

func Open(ctx context.Context, backend backends.Backend) (DB, error) {
	db := newDB(backend)
	return db, db.load(ctx)
}

func newDB(backend backends.Backend) *dbImpl {
	return &dbImpl{
		backend:  backend,
		existing: map[string]*manifest.Stream{},
		new:      map[string]*manifest.Stream{},
		source:   map[string]string{},
	}
}

func New(backend backends.Backend) DB {
	return newDB(backend)
}

func (d *dbImpl) load(ctx context.Context) error {
	var paths []string
	err := utils.SortedList(ctx, d.backend, HashPrefix,
		func(ctx context.Context, path string) error {
			r, err := d.backend.Get(ctx, path, 0, -1)
			if err != nil {
				return err
			}
			defer r.Close()

			paths = append(paths, path)
			return d.loadStream(ctx, r, path)
		})
	if err != nil {
		return err
	}
	d.paths = paths
	return nil
}

func (d *dbImpl) loadStream(ctx context.Context, stream io.Reader, path string) error {
	// TODO: reduce code duplication with pathdb.load
	v := make([]byte, len([]byte(versionHeader)))
	_, err := io.ReadFull(stream, v)
	if err != nil {
		if err == io.EOF {
			err = errs.New("unexpected EOF for hashset %q", path)
		}
		return err
	}
	if versionHeader != string(v) {
		return errs.New("invalid hashset version for hashset %q", path)
	}

	r, err := zlib.NewReader(utils.NewUnframingReader(stream))
	if err != nil {
		return errs.New("error for hashset %q: %+v", path, err)
	}
	defer func() {
		err = errs.Combine(err, r.Close())
	}()

	for {
		var set manifest.HashSet
		err := utils.UnmarshalSized(r, &set)
		if err != nil {
			if err == io.EOF {
				break
			}
			return errs.New("error for hashset %q: %+v", path, err)
		}
		for _, entry := range set.Hashes {
			hashBytes := entry.Hash
			if len(hashBytes) != sha256.Size {
				if len(hashBytes) != sha256.Size*2 {
					return errs.New("unknown hash length for hashset %q", path)
				}
				// TODO: sadlol, remove after everything is migrated
				hashBytes = hashBytes[len(hashBytes)-sha256.Size:]
			}

			hash := string(hashBytes)
			// TODO: log on overwrites?
			d.existing[hash] = entry.Data
			d.source[hash] = path
		}
	}

	return nil
}

func (d *dbImpl) Has(ctx context.Context, hash string) (exists bool, err error) {
	stream, err := d.Lookup(ctx, hash)
	return stream != nil, err
}

func (d *dbImpl) Lookup(ctx context.Context, hash string) (*manifest.Stream, error) {
	rv := d.existing[hash]
	if rv != nil {
		return rv, nil
	}
	return d.new[hash], nil
}

func (d *dbImpl) Put(ctx context.Context, hash string, data *manifest.Stream) error {
	d.new[hash] = data
	return nil
}

func (d *dbImpl) flush(ctx context.Context, hashes map[string]*manifest.Stream) (string, error) {
	if len(hashes) == 0 {
		return "", nil
	}

	// TODO: LSM tree instead of this
	var set manifest.HashSet
	for hash, data := range hashes {
		set.Hashes = append(set.Hashes, &manifest.HashedData{
			Hash: []byte(hash),
			Data: data,
		})
	}

	// TODO: reduce code duplication with pathdb.Serialize
	data, err := utils.MarshalSized(&set)
	if err != nil {
		return "", err
	}

	var out bytes.Buffer
	compressor := zlib.NewWriter(&out)
	_, err = compressor.Write(data)
	if err != nil {
		return "", err
	}
	err = compressor.Close()
	if err != nil {
		return "", err
	}

	path := HashPrefix + streams.IdPathComponent(utils.IdGen())
	if out.Len() <= SmallHashsetThreshold {
		path += SmallHashsetSuffix
	}

	err = d.backend.Put(ctx, path, io.MultiReader(
		bytes.NewReader([]byte(versionHeader)),
		utils.NewFramingReader(&out)))
	if err != nil {
		return "", err
	}

	return path, nil
}

func (d *dbImpl) Flush(ctx context.Context) error {
	path, err := d.flush(ctx, d.new)
	if err != nil {
		return err
	}
	for hash, data := range d.new {
		d.existing[hash] = data
		d.source[hash] = path
	}
	d.new = map[string]*manifest.Stream{}

	return nil
}

func (d *dbImpl) Close() error {
	return nil
}

func (d *dbImpl) Iterate(ctx context.Context, cb func(ctx context.Context, hash, hashset string, data *manifest.Stream) error) error {
	for hash, data := range d.existing {
		err := cb(ctx, hash, d.source[hash], data)
		if err != nil {
			return err
		}
	}
	for hash, data := range d.new {
		err := cb(ctx, hash, "", data)
		if err != nil {
			return err
		}
	}
	return nil
}
