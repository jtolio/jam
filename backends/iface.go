package backends

import (
	"context"
	"io"
)

// Backend is a simplistic interface for storing data. The goal is to make as
// simple as possible of an interface, so Jam will work with as many backend
// providers as possible.
type Backend interface {
	// Get takes a path and an offset and returns an io.ReadCloser consisting of
	// data from the offset to the end of the object. The offset will be >= 0 and
	// less than the object's length. Behavior outside of those bounds is undefined.
	Get(ctx context.Context, path string, offset int64) (io.ReadCloser, error)
	// Put creates a new object at path consisting of the provided data.
	// Put will not be called if the path exists, so behavior for existent paths
	// is undefined. Put may be called with a path with forward-slash delimiters.
	Put(ctx context.Context, path string, data io.Reader) error
	// Delete removes the object at path. Delete will not be called if the path
	// doesn't exist, so behavior for nonexistent paths is undefined.
	Delete(ctx context.Context, path string) error
	// List should call 'cb' for all paths (recursively) starting with prefix
	// until cb returns. 'prefix' will either be empty or end with a
	// forward-slash. It is not required for List to return paths in order.
	List(ctx context.Context, prefix string, cb func(path string) error) error
}
