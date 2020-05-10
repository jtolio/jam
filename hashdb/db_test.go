package hashdb

import (
	"context"
	"io/ioutil"
	"net/url"
	"os"
	"testing"

	"github.com/jtolds/jam/backends/fs"
	"github.com/jtolds/jam/manifest"
	"github.com/stretchr/testify/require"
)

var ctx = context.Background()

func TestHashDB(t *testing.T) {
	td, err := ioutil.TempDir("", "hashdbtest")
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

	stream, err := db.Lookup(ctx, "a")
	require.NoError(t, err)
	require.Nil(t, stream)
	stream, err = db.Lookup(ctx, "b")
	require.NoError(t, err)
	require.Nil(t, stream)

	require.NoError(t, db.Put(ctx, "a", &manifest.Stream{Ranges: []*manifest.Range{
		{Blob: "1", Offset: 0, Length: 1},
	}}))
	require.NoError(t, db.Flush(ctx))

	require.NoError(t, db.Close())

	db, err = Open(ctx, b)
	require.NoError(t, err)

	stream, err = db.Lookup(ctx, "a")
	require.NoError(t, err)
	require.Equal(t, len(stream.Ranges), 1)
	require.Equal(t, stream.Ranges[0].Blob, "1")
	require.Equal(t, stream.Ranges[0].Offset, int64(0))
	require.Equal(t, stream.Ranges[0].Length, int64(1))
	stream, err = db.Lookup(ctx, "b")
	require.NoError(t, err)
	require.Nil(t, stream)

	require.NoError(t, db.Put(ctx, "b", &manifest.Stream{Ranges: []*manifest.Range{
		{Blob: "2", Offset: 4, Length: 2},
		{Blob: "3", Offset: 1, Length: 3},
	}}))
	require.NoError(t, db.Flush(ctx))

	require.NoError(t, db.Close())

	db, err = Open(ctx, b)
	require.NoError(t, err)

	stream, err = db.Lookup(ctx, "a")
	require.NoError(t, err)
	require.Equal(t, len(stream.Ranges), 1)
	require.Equal(t, stream.Ranges[0].Blob, "1")
	require.Equal(t, stream.Ranges[0].Offset, int64(0))
	require.Equal(t, stream.Ranges[0].Length, int64(1))
	stream, err = db.Lookup(ctx, "b")
	require.NoError(t, err)
	require.Equal(t, len(stream.Ranges), 2)
	require.Equal(t, stream.Ranges[0].Blob, "2")
	require.Equal(t, stream.Ranges[0].Offset, int64(4))
	require.Equal(t, stream.Ranges[0].Length, int64(2))
	require.Equal(t, stream.Ranges[1].Blob, "3")
	require.Equal(t, stream.Ranges[1].Offset, int64(1))
	require.Equal(t, stream.Ranges[1].Length, int64(3))

	require.NoError(t, db.Close())
}
