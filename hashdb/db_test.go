package hashdb

import (
	"context"
	"crypto/sha256"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/jtolio/jam/backends/fs"
	"github.com/jtolio/jam/manifest"
	"github.com/jtolio/jam/utils"
)

var ctx = context.Background()

func extendHash(hash string) string {
	return strings.Repeat("\x00", sha256.Size-len(hash)) + hash
}

func TestHashDB(t *testing.T) {
	td, err := os.MkdirTemp("", "hashdbtest")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, os.RemoveAll(td))
	}()

	b, err := fs.New(ctx, &url.URL{Path: td})
	require.NoError(t, err)
	defer func() {
		require.NoError(t, b.Close())
	}()

	db, err := Open(ctx, b)
	require.NoError(t, err)

	stream, err := db.Lookup(ctx, extendHash("a"))
	require.NoError(t, err)
	require.Nil(t, stream)
	stream, err = db.Lookup(ctx, extendHash("b"))
	require.NoError(t, err)
	require.Nil(t, stream)

	require.NoError(t, db.Put(ctx, extendHash("a"),
		&manifest.Stream{Ranges: []*manifest.Range{
			{BlobBytes: []byte("1"), Offset: 0, Length: 1},
		}}))
	require.NoError(t, db.Flush(ctx))

	require.NoError(t, db.Close())

	db, err = Open(ctx, b)
	require.NoError(t, err)

	stream, err = db.Lookup(ctx, extendHash("a"))
	require.NoError(t, err)
	require.Equal(t, len(stream.Ranges), 1)
	require.Equal(t, stream.Ranges[0].Blob(), utils.PathSafeIdEncode([]byte("1")))
	require.Equal(t, stream.Ranges[0].Offset, int64(0))
	require.Equal(t, stream.Ranges[0].Length, int64(1))
	stream, err = db.Lookup(ctx, extendHash("b"))
	require.NoError(t, err)
	require.Nil(t, stream)

	require.NoError(t, db.Put(ctx, extendHash("b"),
		&manifest.Stream{Ranges: []*manifest.Range{
			{BlobBytes: []byte("2"), Offset: 4, Length: 2},
			{BlobBytes: []byte("3"), Offset: 1, Length: 3},
		}}))
	require.NoError(t, db.Flush(ctx))

	require.NoError(t, db.Close())

	db, err = Open(ctx, b)
	require.NoError(t, err)

	stream, err = db.Lookup(ctx, extendHash("a"))
	require.NoError(t, err)
	require.Equal(t, len(stream.Ranges), 1)
	require.Equal(t, stream.Ranges[0].Blob(), utils.PathSafeIdEncode([]byte("1")))
	require.Equal(t, stream.Ranges[0].Offset, int64(0))
	require.Equal(t, stream.Ranges[0].Length, int64(1))
	stream, err = db.Lookup(ctx, extendHash("b"))
	require.NoError(t, err)
	require.Equal(t, len(stream.Ranges), 2)
	require.Equal(t, stream.Ranges[0].Blob(), utils.PathSafeIdEncode([]byte("2")))
	require.Equal(t, stream.Ranges[0].Offset, int64(4))
	require.Equal(t, stream.Ranges[0].Length, int64(2))
	require.Equal(t, stream.Ranges[1].Blob(), utils.PathSafeIdEncode([]byte("3")))
	require.Equal(t, stream.Ranges[1].Offset, int64(1))
	require.Equal(t, stream.Ranges[1].Length, int64(3))

	require.NoError(t, db.Close())
}
