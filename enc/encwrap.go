package enc

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"

	"github.com/jtolds/jam/backends"
)

// EncWrapper wraps a Backend with encryption.
type EncWrapper struct {
	enc     Codec
	keyGen  KeyGenerator
	backend backends.Backend
}

var _ backends.Backend = (*EncWrapper)(nil)

// NewEncWrapper returns a new Backend with the provided encryption
func NewEncWrapper(encryption Codec, keyGen KeyGenerator, backend backends.Backend) *EncWrapper {
	return &EncWrapper{
		enc:     encryption,
		keyGen:  keyGen,
		backend: backend,
	}
}

// Get implements the Backend interface
func (e *EncWrapper) Get(ctx context.Context, path string, offset, length int64) (io.ReadCloser, error) {
	// See implementation note in List

	// calculate how much back we have to get to get the block that contains the requested offset
	decodedBlockSize := int64(e.enc.DecodedBlockSize())
	encodedBlockSize := int64(e.enc.EncodedBlockSize())
	firstBlock := offset / decodedBlockSize
	encodedLength := int64(-1)
	encodedOffset := firstBlock * encodedBlockSize
	if length > 0 {
		lastByte := offset + length - 1
		blockOfLastByte := lastByte / decodedBlockSize
		encodedLength = (blockOfLastByte+1)*encodedBlockSize - encodedOffset
	}
	fh, err := e.backend.Get(ctx, path, encodedOffset, encodedLength)
	if err != nil {
		return nil, err
	}

	// Implementation note:
	// Keys are always the same for the same path. There is normally a huge risk of nonce reuse
	// in this scenario except it is guaranteed that for backends, a given path will always
	// have the exact same data.
	key := e.keyGen.KeyForPath(path)
	r := DecodeReader(fh, e.enc, &key, firstBlock)

	// we had to rewind to get the enclosing block beginning. now fast forward to skip the
	// initial block bytes.
	if skip := offset - firstBlock*decodedBlockSize; skip > 0 {
		_, err = io.CopyN(ioutil.Discard, r, skip)
		if err != nil {
			fh.Close()
			if err == io.EOF {
				return ioutil.NopCloser(bytes.NewReader(nil)), nil
			}
			return nil, err
		}
	}

	return struct {
		io.Reader
		io.Closer
	}{
		Reader: r,
		Closer: fh,
	}, nil
}

// Put implements the Backend interface
func (e *EncWrapper) Put(ctx context.Context, path string, data io.Reader) error {
	// See implementation note in List
	// See implementation note in Get
	key := e.keyGen.KeyForPath(path)
	return e.backend.Put(ctx, path,
		EncodeReader(
			&padding{r: data, bs: e.enc.DecodedBlockSize()},
			e.enc, &key, 0))
}

// Delete implements the Backend interface
func (e *EncWrapper) Delete(ctx context.Context, path string) error {
	// See implementation note in List
	return e.backend.Delete(ctx, path)
}

// List implements the Backend interface
func (e *EncWrapper) List(ctx context.Context, prefix string,
	cb func(ctx context.Context, path string) error) error {
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

type padding struct {
	r   io.Reader
	bs  int
	n   int64
	pad io.Reader
}

func (pd *padding) Read(p []byte) (n int, err error) {
	if pd.pad != nil {
		return pd.pad.Read(p)
	}
	n, err = pd.r.Read(p)
	pd.n += int64(n)
	if err != io.EOF {
		return n, err
	}
	if pd.n%int64(pd.bs) == 0 {
		pd.pad = bytes.NewReader(nil)
	} else {
		pd.pad = bytes.NewReader(make([]byte, int64(pd.bs)-pd.n%int64(pd.bs)))
	}
	if n == 0 {
		return pd.Read(p)
	}
	return n, nil
}

func (e *EncWrapper) Close() error {
	return e.backend.Close()
}
