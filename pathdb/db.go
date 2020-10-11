package pathdb

import (
	"bytes"
	"compress/zlib"
	"context"
	"crypto/sha256"
	"io"
	"io/ioutil"
	"reflect"
	"regexp"
	"strings"

	"github.com/zeebo/errs"

	"github.com/jtolds/jam/backends"
	"github.com/jtolds/jam/blobs"
	"github.com/jtolds/jam/manifest"
	"github.com/jtolds/jam/pathdb/b"
	"github.com/jtolds/jam/streams"
	"github.com/jtolds/jam/utils"
)

const versionHeader = "jam-v0\n"

type DB struct {
	backend backends.Backend
	blobs   *blobs.Store
	changed bool
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
		err := utils.UnmarshalSized(r, &page)
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
				contentHash := entry.Content.Hash
				if len(contentHash) > 0 && len(contentHash) != sha256.Size {
					if len(contentHash) != sha256.Size*2 {
						return errs.New("unknown hash length")
					}
					// TODO: sadlol, remove after everything is migrated
					entry.Content.Hash = contentHash[len(contentHash)-sha256.Size:]
				}
				db.tree.Set(string(entry.Path), entry.Content)
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

func (db *DB) HasPrefix(ctx context.Context, prefix string) (exists bool, err error) {
	it, _ := db.tree.Seek(prefix)
	path, _, err := it.Next()
	if err != nil {
		if err == io.EOF {
			return false, nil
		}
		return false, err
	}
	return strings.HasPrefix(path, prefix), nil
}

func (db *DB) List(ctx context.Context, prefix string, recursive bool,
	cb func(ctx context.Context, path string, content *manifest.Content) error) error {
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

		if !recursive {
			if idx := strings.Index(path[len(prefix):], "/"); idx >= 0 {
				path = path[:len(prefix)+idx]
				err = cb(ctx, path, nil)
				if err != nil {
					return err
				}
				it, _ = db.tree.Seek(path + "0") // "0" is the next byte after "/"
				continue
			}
		}

		err = cb(ctx, path, content)
		if err != nil {
			return err
		}
	}
	return nil
}

type PutState int

const (
	PutStateNew       PutState = 1
	PutStateChanged   PutState = 2
	PutStateUnchanged PutState = 3
)

func (db *DB) Put(ctx context.Context, path string, content *manifest.Content) (state PutState, err error) {
	state = PutStateNew
	if v, ok := db.tree.Get(path); ok {
		if reflect.DeepEqual(v, content) {
			return PutStateUnchanged, nil
		}
		state = PutStateChanged
	}

	db.tree.Set(path, content)
	db.changed = true
	return state, nil
}

func (db *DB) Delete(ctx context.Context, path string) (removed bool, err error) {
	if _, ok := db.tree.Get(path); !ok {
		return false, nil
	}

	utils.L(ctx).Normalf("deleted path %q", path)
	db.tree.Delete(path)
	db.changed = true
	return true, nil
}

// Rename renames paths using regexp.ReplaceAllString (replacement can have
// regexp expansions). See the docs for regexp.ReplaceAllString
func (db *DB) Rename(ctx context.Context, re *regexp.Regexp, replacement string) (renamed int, err error) {
	type element struct {
		path    string
		content *manifest.Content
	}
	var queue []element
	it, err := db.tree.SeekFirst()
	if err != nil {
		if err == io.EOF {
			return 0, nil
		}
		return 0, err
	}
	defer it.Close()

	for {
		path, content, err := it.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return 0, err
		}
		if re.MatchString(path) {
			queue = append(queue, element{path: path, content: content})
		}
	}

	for _, el := range queue {
		db.tree.Delete(el.path)
	}

	for _, el := range queue {
		db.tree.Set(re.ReplaceAllString(el.path, replacement), el.content)
	}

	if len(queue) > 0 {
		db.changed = true
	}

	return len(queue), nil
}

func (db *DB) DeleteAll(ctx context.Context, matcher func(path string) (delete bool)) (removed int, err error) {
	var queue []string
	it, err := db.tree.SeekFirst()
	if err != nil {
		if err == io.EOF {
			return 0, nil
		}
		return 0, err
	}
	defer it.Close()

	for {
		path, _, err := it.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return 0, err
		}
		if matcher(path) {
			queue = append(queue, path)
		}
	}

	for _, el := range queue {
		db.tree.Delete(el)
	}
	if len(queue) > 0 {
		db.changed = true
	}

	return len(queue), nil
}

func (db *DB) Changed() bool {
	return db.changed
}

func (db *DB) SerializeTo(ctx context.Context, destinationPath string) error {
	// TODO: don't just dump all of the entries into the root manifest page.
	// TODO: even if the whole manifest is in RAM, don't double the RAM usage here
	var entries manifest.EntrySet

	it, err := db.tree.SeekFirst()
	if err != nil {
		if err != io.EOF {
			return err
		}
	} else {
		defer it.Close()
		for {
			path, content, err := it.Next()
			if err != nil {
				if err == io.EOF {
					break
				}
				return err
			}
			entries.Entries = append(entries.Entries, &manifest.Entry{
				Path:    []byte(path),
				Content: content,
			})
		}
	}

	var page manifest.Page
	page.Descendents = &manifest.Page_Entries{
		Entries: &entries,
	}
	data, err := utils.MarshalSized(&page)
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

	return db.backend.Put(ctx, destinationPath, ioutil.NopCloser(io.MultiReader(
		bytes.NewReader([]byte(versionHeader)),
		utils.NewFramingReader(&out))))
}

func (db *DB) Close() error {
	tree := db.tree
	db.tree = nil
	if tree != nil {
		tree.Close()
	}
	return nil
}
