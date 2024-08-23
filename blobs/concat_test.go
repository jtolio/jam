package blobs

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/jtolio/jam/manifest"
)

var ctx = context.Background()

func TestConcatEmpty(t *testing.T) {
	var calls int
	c := newConcat(&entry{
		source: io.NopCloser(bytes.NewReader([]byte{})),
		cb: func(ctx context.Context, m *manifest.Stream, lastOfBlob bool) error {
			calls++
			require.Equal(t, 1, calls)
			require.Equal(t, 0, len(m.Ranges))
			require.True(t, lastOfBlob)
			return nil
		},
	})

	data, err := io.ReadAll(c)
	require.NoError(t, err)
	require.True(t, bytes.Equal(data, nil))
	require.NoError(t, c.Cut(ctx))
}

func TestConcatSimple(t *testing.T) {
	var streams []*manifest.Stream
	defer func() {
		require.Equal(t, 1, len(streams))
		require.Equal(t, 1, len(streams[0].Ranges))
	}()
	c := newConcat(&entry{
		source: io.NopCloser(bytes.NewReader([]byte("hello"))),
		cb: func(ctx context.Context, m *manifest.Stream, lastOfBlob bool) error {
			streams = append(streams, m)
			require.Equal(t, int64(0), m.Ranges[0].Offset)
			require.Equal(t, int64(5), m.Ranges[0].Length)
			require.True(t, lastOfBlob)
			return nil
		},
	})

	data, err := io.ReadAll(c)
	require.NoError(t, err)
	require.True(t, bytes.Equal(data, []byte("hello")))
	require.NoError(t, c.Cut(ctx))
}

func TestConcatTwo(t *testing.T) {
	var streams []*manifest.Stream
	defer func() {
		require.Equal(t, 2, len(streams))
		require.Equal(t, 1, len(streams[0].Ranges))
		require.Equal(t, 1, len(streams[1].Ranges))
	}()
	var calls1, calls2 int
	c := newConcat(&entry{
		source: io.NopCloser(bytes.NewReader([]byte("hello "))),
		cb: func(ctx context.Context, m *manifest.Stream, lastOfBlob bool) error {
			calls1++
			streams = append(streams, m)
			require.Equal(t, 1, calls1)
			require.Equal(t, int64(0), m.Ranges[0].Offset)
			require.Equal(t, int64(6), m.Ranges[0].Length)
			require.False(t, lastOfBlob)
			return nil
		},
	}, &entry{
		source: io.NopCloser(bytes.NewReader([]byte("world!"))),
		cb: func(ctx context.Context, m *manifest.Stream, lastOfBlob bool) error {
			calls2++
			streams = append(streams, m)
			require.Equal(t, 1, calls2)
			require.Equal(t, int64(6), m.Ranges[0].Offset)
			require.Equal(t, int64(6), m.Ranges[0].Length)
			require.True(t, lastOfBlob)
			return nil
		},
	})

	data, err := io.ReadAll(c)
	require.NoError(t, err)
	require.True(t, bytes.Equal(data, []byte("hello world!")))
	require.NoError(t, c.Cut(ctx))
}

func TestConcatTwoCutBefore(t *testing.T) {
	var streams []*manifest.Stream
	defer func() {
		require.Equal(t, 2, len(streams))
		require.Equal(t, 2, len(streams[0].Ranges))
		require.Equal(t, 1, len(streams[1].Ranges))
	}()
	var calls1, calls2 int
	c := newConcat(&entry{
		source: io.NopCloser(bytes.NewReader([]byte("hello "))),
		cb: func(ctx context.Context, m *manifest.Stream, lastOfBlob bool) error {
			calls1++
			streams = append(streams, m)
			require.Equal(t, 1, calls1)
			require.Equal(t, 2, len(m.Ranges))
			require.Equal(t, int64(0), m.Ranges[0].Offset)
			require.Equal(t, int64(4), m.Ranges[0].Length)
			require.Equal(t, int64(0), m.Ranges[1].Offset)
			require.Equal(t, int64(2), m.Ranges[1].Length)
			require.NotEqual(t, m.Ranges[0].Blob(), m.Ranges[1].Blob())
			require.False(t, lastOfBlob)
			return nil
		},
	}, &entry{
		source: io.NopCloser(bytes.NewReader([]byte("world!"))),
		cb: func(ctx context.Context, m *manifest.Stream, lastOfBlob bool) error {
			calls2++
			streams = append(streams, m)
			require.Equal(t, 1, calls2)
			require.Equal(t, 1, len(m.Ranges))
			require.Equal(t, int64(2), m.Ranges[0].Offset)
			require.Equal(t, int64(6), m.Ranges[0].Length)
			require.True(t, lastOfBlob)
			return nil
		},
	})

	var buf [4]byte
	_, err := io.ReadFull(c, buf[:])
	require.NoError(t, err)
	require.True(t, bytes.Equal(buf[:], []byte("hell")))
	require.NoError(t, c.Cut(ctx))

	data, err := io.ReadAll(c)
	require.NoError(t, err)
	require.True(t, bytes.Equal(data, []byte("o world!")))
	require.NoError(t, c.Cut(ctx))
}

func TestConcatTwoCutOn(t *testing.T) {
	var streams []*manifest.Stream
	defer func() {
		require.Equal(t, 2, len(streams))
		require.Equal(t, 1, len(streams[0].Ranges))
		require.Equal(t, 1, len(streams[1].Ranges))
	}()
	var calls1, calls2 int
	c := newConcat(&entry{
		source: io.NopCloser(bytes.NewReader([]byte("hello "))),
		cb: func(ctx context.Context, m *manifest.Stream, lastOfBlob bool) error {
			calls1++
			streams = append(streams, m)
			require.Equal(t, 1, calls1)
			require.Equal(t, int64(0), m.Ranges[0].Offset)
			require.Equal(t, int64(6), m.Ranges[0].Length)
			require.False(t, lastOfBlob)
			return nil
		},
	}, &entry{
		source: io.NopCloser(bytes.NewReader([]byte("world!"))),
		cb: func(ctx context.Context, m *manifest.Stream, lastOfBlob bool) error {
			calls2++
			streams = append(streams, m)
			require.Equal(t, 1, calls2)
			require.Equal(t, int64(0), m.Ranges[0].Offset)
			require.Equal(t, int64(6), m.Ranges[0].Length)
			require.True(t, lastOfBlob)
			return nil
		},
	})

	var buf [6]byte
	_, err := io.ReadFull(c, buf[:])
	require.NoError(t, err)
	require.True(t, bytes.Equal(buf[:], []byte("hello ")))
	require.NoError(t, c.Cut(ctx))

	data, err := io.ReadAll(c)
	require.NoError(t, err)
	require.True(t, bytes.Equal(data, []byte("world!")))
	require.NoError(t, c.Cut(ctx))
}

func TestConcatTwoCutAfter(t *testing.T) {
	var streams []*manifest.Stream
	defer func() {
		require.Equal(t, 2, len(streams))
		require.Equal(t, 1, len(streams[0].Ranges))
		require.Equal(t, 2, len(streams[1].Ranges))
	}()
	var calls1, calls2 int
	c := newConcat(&entry{
		source: io.NopCloser(bytes.NewReader([]byte("hello "))),
		cb: func(ctx context.Context, m *manifest.Stream, lastOfBlob bool) error {
			calls1++
			streams = append(streams, m)
			require.Equal(t, 1, calls1)
			require.Equal(t, 1, len(m.Ranges))
			require.Equal(t, int64(0), m.Ranges[0].Offset)
			require.Equal(t, int64(6), m.Ranges[0].Length)
			require.True(t, lastOfBlob)
			return nil
		},
	}, &entry{
		source: io.NopCloser(bytes.NewReader([]byte("world!"))),
		cb: func(ctx context.Context, m *manifest.Stream, lastOfBlob bool) error {
			calls2++
			streams = append(streams, m)
			require.Equal(t, 1, calls2)
			require.Equal(t, 2, len(m.Ranges))
			require.Equal(t, int64(6), m.Ranges[0].Offset)
			require.Equal(t, int64(2), m.Ranges[0].Length)
			require.Equal(t, int64(0), m.Ranges[1].Offset)
			require.Equal(t, int64(4), m.Ranges[1].Length)
			require.NotEqual(t, m.Ranges[0].Blob(), m.Ranges[1].Blob())
			require.True(t, lastOfBlob)
			return nil
		},
	})

	var buf [8]byte
	_, err := io.ReadFull(c, buf[:])
	require.NoError(t, err)
	require.True(t, bytes.Equal(buf[:], []byte("hello wo")))
	require.NoError(t, c.Cut(ctx))

	data, err := io.ReadAll(c)
	require.NoError(t, err)
	require.True(t, bytes.Equal(data, []byte("rld!")))
	require.NoError(t, c.Cut(ctx))
}
