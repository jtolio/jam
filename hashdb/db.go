package hashdb

import (
	"bytes"
	"compress/zlib"
	"context"
	"io"

	"github.com/jtolds/jam/backends"
	"github.com/jtolds/jam/blobs"
	"github.com/jtolds/jam/manifest"
	"github.com/jtolds/jam/utils"
	"github.com/zeebo/errs"
)

const versionHeader = "jam-v0\n"
const hashPrefix = "hash/"

type DB struct {
	backend      backends.Backend
	maxUnflushed int

	// TODO: do an LSM tree instead of putting all of this in RAM
	existing map[string]*manifest.Stream
	new      map[string]*manifest.Stream
}

func Open(ctx context.Context, backend backends.Backend, maxUnflushed int) (*DB, error) {
	db := New(backend, maxUnflushed)
	return db, db.load(ctx)
}

func New(backend backends.Backend, maxUnflushed int) *DB {
	return &DB{
		backend:      backend,
		maxUnflushed: maxUnflushed,
		existing:     map[string]*manifest.Stream{},
		new:          map[string]*manifest.Stream{},
	}
}

func (d *DB) load(ctx context.Context) error {
	return d.backend.List(ctx, hashPrefix,
		func(ctx context.Context, path string) error {
			r, err := d.backend.Get(ctx, path, 0, -1)
			if err != nil {
				return err
			}
			defer r.Close()

			return d.loadStream(ctx, r)
		})
}

func (d *DB) loadStream(ctx context.Context, stream io.Reader) error {
	// TODO: reduce code duplication with pathdb.load
	v := make([]byte, len([]byte(versionHeader)))
	_, err := io.ReadFull(stream, v)
	if err != nil {
		if err == io.EOF {
			err = errs.Wrap(io.ErrUnexpectedEOF)
		}
		return err
	}
	if versionHeader != string(v) {
		return errs.New("invalid manifest version")
	}

	r, err := zlib.NewReader(utils.NewUnframingReader(stream))
	if err != nil {
		return err
	}
	defer func() {
		err = errs.Combine(err, r.Close())
	}()

	for {
		var set manifest.HashSet
		err := manifest.UnmarshalSized(r, &set)
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		for _, entry := range set.Hashes {
			// TODO: log on overwrites
			d.existing[string(entry.Hash)] = entry.Data
		}
	}

	return nil
}

func (d *DB) Has(ctx context.Context, hash string) (exists bool, err error) {
	stream, err := d.Lookup(ctx, hash)
	return stream != nil, err
}

func (d *DB) Lookup(ctx context.Context, hash string) (*manifest.Stream, error) {
	rv := d.existing[hash]
	if rv != nil {
		return rv, nil
	}
	return d.new[hash], nil
}

func (d *DB) Put(ctx context.Context, hash string, data *manifest.Stream) error {
	d.new[hash] = data
	if len(d.new) <= d.maxUnflushed {
		return nil
	}
	return d.Flush(ctx)
}

func (d *DB) Flush(ctx context.Context) error {
	// TODO: LSM tree instead of this
	var set manifest.HashSet
	for hash, data := range d.new {
		set.Hashes = append(set.Hashes, &manifest.HashedData{
			Hash: []byte(hash),
			Data: data,
		})
	}

	// TODO: reduce code duplication with pathdb.Serialize
	data, err := manifest.MarshalSized(&set)
	if err != nil {
		return err
	}

	var out bytes.Buffer
	compressor := zlib.NewWriter(&out)
	_, err = compressor.Write(data)
	if err != nil {
		return err
	}
	err = compressor.Close()
	if err != nil {
		return err
	}

	err = d.backend.Put(ctx, hashPrefix+blobs.IdGen(), io.MultiReader(
		bytes.NewReader([]byte(versionHeader)),
		utils.NewFramingReader(&out)))
	if err != nil {
		return err
	}

	for hash, data := range d.new {
		d.existing[hash] = data
	}
	d.new = map[string]*manifest.Stream{}

	return nil
}

func (d *DB) Close() error {
	return nil
}