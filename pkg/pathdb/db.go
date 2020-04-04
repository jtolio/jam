package pathdb

import (
	"bytes"
	"compress/zlib"
	"context"
	"io"
	"io/ioutil"
	"strings"

	"github.com/zeebo/errs"

	"github.com/jtolds/jam/backends"
	"github.com/jtolds/jam/pkg/blobs"
	"github.com/jtolds/jam/pkg/manifest"
	"github.com/jtolds/jam/pkg/pathdb/b"
	"github.com/jtolds/jam/pkg/streams"
	"github.com/jtolds/jam/pkg/utils"
)

const versionHeader = "jam-v0\n"

type DB struct {
	backend backends.Backend
	blobs   *blobs.Store
	// TODO: don't expect the whole manifest to fit into RAM
	tree *b.Tree
}

func Open(ctx context.Context, backend backends.Backend, blobStore *blobs.Store, stream io.Reader) (*DB, error) {
	db := New(backend, blobStore)
	return db, db.load(ctx, stream)
}

func New(backend backends.Backend, blobStore *blobs.Store) *DB {
	return &DB{
		backend: backend,
		blobs:   blobStore,
		tree:    b.TreeNew(strings.Compare),
	}
}

func (db *DB) load(ctx context.Context, stream io.Reader) error {
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
		var page manifest.Page
		err := manifest.UnmarshalSized(r, &page)
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		if branch := page.GetBranch(); branch != nil {
			substream, err := streams.Open(ctx, db.backend, branch)
			if err != nil {
				return err
			}
			err = db.load(ctx, substream)
			if err != nil {
				return err
			}
		}
		if entries := page.GetEntries(); entries != nil {
			for _, entry := range entries.Entries {
				db.tree.Set(entry.Path, entry.Content)
			}
		}
	}

	return nil
}

func (db *DB) Get(ctx context.Context, path string) (*manifest.Content, error) {
	v, ok := db.tree.Get(path)
	if !ok {
		return nil, nil
	}
	return v, nil
}

func (db *DB) List(ctx context.Context, prefix, delimiter string,
	cb func(ctx context.Context, path string, content *manifest.Content) error) error {
	lastPath := ""
	var lastContent *manifest.Content
	lastSet := false
	it, _ := db.tree.Seek(prefix)
	for {
		path, content, err := it.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		if !strings.HasPrefix(path, prefix) {
			break
		}

		if delimiter != "" {
			// TODO: we should skip the iterator forward for performance,
			if idx := strings.Index(path[len(prefix):], delimiter); idx >= 0 {
				path = path[:len(prefix)+idx]
				content = nil
			}
		}

		if !lastSet || path != lastPath || content != lastContent {
			// TODO: once we skip the iterator forward, we can stop doing
			// this deduplication business
			err = cb(ctx, path, content)
			if err != nil {
				return err
			}
			lastPath = path
			lastContent = content
			lastSet = true
		}
	}
	return nil
}

func (db *DB) Put(ctx context.Context, path string, content *manifest.Content) error {
	db.tree.Set(path, content)
	return nil
}

func (db *DB) Delete(ctx context.Context, path string) error {
	db.tree.Delete(path)
	return nil
}

func (db *DB) Serialize(ctx context.Context) (io.ReadCloser, error) {
	// TODO: don't just dump all of the entries into the root manifest page.
	// TODO: even if the whole manifest is in RAM, don't double the RAM usage here
	var entries manifest.EntrySet

	it, err := db.tree.SeekFirst()
	if err != nil {
		if err != io.EOF {
			return nil, err
		}
	} else {
		defer it.Close()
		for {
			path, content, err := it.Next()
			if err != nil {
				if err == io.EOF {
					break
				}
				return nil, err
			}
			entries.Entries = append(entries.Entries, &manifest.Entry{
				Path:    path,
				Content: content,
			})
		}
	}

	var page manifest.Page
	page.Descendents = &manifest.Page_Entries{
		Entries: &entries,
	}
	data, err := manifest.MarshalSized(&page)
	if err != nil {
		return nil, err
	}

	var out bytes.Buffer
	compressor := zlib.NewWriter(&out)
	_, err = compressor.Write(data)
	if err != nil {
		return nil, err
	}
	err = compressor.Close()
	if err != nil {
		return nil, err
	}

	return ioutil.NopCloser(io.MultiReader(
		bytes.NewReader([]byte(versionHeader)),
		utils.NewFramingReader(&out))), nil
}

func (db *DB) Close() error {
	tree := db.tree
	db.tree = nil
	if tree != nil {
		tree.Close()
	}
	return nil
}
