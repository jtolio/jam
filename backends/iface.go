package backends

import (
	"context"
	"errors"
	"io"
)

var (
	ErrNotExist = errors.New("error: object doesn't exist")
)

// Backend is a simplistic interface for storing data. The goal is to make as
// simple as possible of an interface, so Jam will work with as many backend
// providers as possible.
//
// Invariants:
//  * Objects at paths will not be replaced. They are treated as immutable
//    and will not be changed. Different contents will never be stored at the
//    same path.
//  * Once deleted, a path with not be re-put.
//  * Paths do not contain any private user data. As three examples, paths will
//    be of the form:
//      meta/<timestamp>
//			hash/<base32>/<hash>
//      blob/<base32>/<hash>
//		etc.
//  * The prefix of one path will never be the full path of another object.
//  * A path element (a part of a path separated by forward slashes)
//		will always be alphanumeric.
type Backend interface {
	// Get takes a path and an offset and returns an io.ReadCloser consisting of
	// data from the offset to the end of the object. The offset will be >= 0 and
	// less than the object's true length. If length > 0, then only that many
	// bytes after the offset are requested (but more can be returned). If
	// length is -1, the rest of the object after the offset is requested. Behavior
	// outside any of of these bounds is undefined.
	// Implementors note: due to the above details, length can be ignored and is
	// only provided for potential optimization.
	// If the object doesn't exist, errors.Is(err, ErrNotExist) should return
	// true.
	Get(ctx context.Context, path string, offset, length int64) (io.ReadCloser, error)
	// Put creates a new object at path consisting of the provided data.
	// Object content for a specific path will always be the same, so Puts for
	// the same path can either replace the object at the path or silently
	// return with no failure, leaving the existing data in place.
	// Put may be called with a path with forward-slash delimiters.
	// Put is expected *not* to create a partial object if data returns an error
	// other than io.EOF, and the object should not show up in listing or be
	// returned from Gets until the Put is complete.
	Put(ctx context.Context, path string, data io.Reader) error
	// Delete removes the object at path. If the object is already gone, Delete
	// should return no error.
	Delete(ctx context.Context, path string) error
	// List should call 'cb' for all paths (recursively) starting with prefix
	// until there are no more paths to return or cb returns an error.
	// 'prefix' will either be empty or end with a forward-slash. It is not
	// required for List to return paths in order. It is expected that all paths
	// returned are full paths.
	List(ctx context.Context, prefix string, cb func(ctx context.Context, path string) error) error
	// Close closes the Backend
	Close() error
}
