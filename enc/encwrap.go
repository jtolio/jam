package enc

import (
	"context"
	"fmt"
	"io"

	"github.com/jtolds/jam/backends"
)

// EncWrapper wraps a Backend with encryption.
type EncWrapper struct {
	enc     Encrypter
	backend backends.Backend
}

var _ backends.Backend = (*EncWrapper)(nil)

// NewEncWrapper returns a new Backend with the provided encryption
func NewEncWrapper(enc Encrypter, backend backends.Backend) *EncWrapper {
	return &EncWrapper{
		enc:     enc,
		backend: backend,
	}
}

// Get implements the Backend interface
func (e *EncWrapper) Get(ctx context.Context, path string, offset int64) (io.ReadCloser, error) {
	// See implementation note in List
	return nil, fmt.Errorf("unimplemented")
}

// Put implements the Backend interface
func (e *EncWrapper) Put(ctx context.Context, path string, data io.Reader) error {
	// See implementation note in List
	return fmt.Errorf("unimplemented")
}

// Delete implements the Backend interface
func (e *EncWrapper) Delete(ctx context.Context, path string) error {
	// See implementation note in List
	return e.backend.Delete(ctx, path)
}

// List implements the Backend interface
func (e *EncWrapper) List(ctx context.Context, prefix string, cb func(path string) error) error {
	// Implementation note:
	// Jam has two levels of paths: the paths of the user's data and the paths passed
	// to the backend. User paths are things like "music/pinkfloyd/thewall.mp3", but
	// backend paths are things like "meta/1575238475" or "blob/4a/b09b33b...". As all
	// user data (including user paths) are stored in the backend as content and not
	// as paths, paths are sufficiently devoid of information to require encryption.
	// The backend already has the ability to determine which sorts of paths
	// contain metadata vs real data or to determine upload timestamps.
	// So, paths to the backend are passed through.
	return e.backend.List(ctx, prefix, cb)
}
