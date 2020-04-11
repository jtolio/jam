package backends

import (
	"context"
	"io"
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
//  * Paths do not contain any private user data. As two examples, paths will
//    be of the form:
//      meta/<timestamp>
//      blob/<hex>/<hash>
//		etc.
//  * The prefix of one path will never be the full path of another object.
//
// One other interesting note - backends are allowed to garbage-fill some
// arbitrary length of data at the end of a Get. If a caller does a Put
// for 10 bytes, a Get with a negative length is allowed to return a reader
// for more than 10 bytes, where the bytes after the 10th are arbitrary.
// If a Get with offset 0, length 5 happens on the same 10 byte object,
// the returned Get can return more than 5 bytes (though they won't be
// read). This does mean that readers returned from Gets will most certainly
// be closed before they are exhausted.
type Backend interface {
	// Get takes a path and an offset and returns an io.ReadCloser consisting of
	// data from the offset to the end of the object. The offset will be >= 0 and
	// less than the object's true length. If length > 0, then only that many
	// bytes after the offset are requested (but more can be returned). If
	// length is -1, the rest of the object after the offset is requested. Behavior
	// outside any of of these bounds is undefined.
	// Implementors note: due to the above details, length can be ignored and is
	// only provided for potential optimization.
	Get(ctx context.Context, path string, offset, length int64) (io.ReadCloser, error)
	// Put creates a new object at path consisting of the provided data.
	// Put will not be called if the path exists, so behavior for existent paths
	// is undefined. Put may be called with a path with forward-slash delimiters.
	Put(ctx context.Context, path string, data io.Reader) error
	// Delete removes the object at path. Delete will not be called if the path
	// doesn't exist, so behavior for nonexistent paths is undefined.
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
