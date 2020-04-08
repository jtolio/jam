package blobs

import (
	"bytes"
	"io"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/jtolds/jam/pkg/manifest"
)

func TestConcatEmpty(t *testing.T) {
	var calls int
	c := newConcat(&entry{
		source: ioutil.NopCloser(bytes.NewReader([]byte{})),
		cb: func(m *manifest.Stream) {
			calls++
			require.Equal(t, 1, calls)
			require.Equal(t, 0, len(m.Ranges))
		},
	})

	data, err := ioutil.ReadAll(c)
	require.NoError(t, err)
	require.True(t, bytes.Equal(data, nil))
	c.Cut()
}

func TestConcatSimple(t *testing.T) {
	var calls int
	c := newConcat(&entry{
		source: ioutil.NopCloser(bytes.NewReader([]byte("hello"))),
		cb: func(m *manifest.Stream) {
			calls++
			require.Equal(t, 1, calls)
			require.Equal(t, 1, len(m.Ranges))
			require.Equal(t, int64(0), m.Ranges[0].Offset)
			require.Equal(t, int64(5), m.Ranges[0].Length)
		},
	})

	data, err := ioutil.ReadAll(c)
	require.NoError(t, err)
	require.True(t, bytes.Equal(data, []byte("hello")))
	c.Cut()
}

func TestConcatTwo(t *testing.T) {
	var calls1, calls2 int
	c := newConcat(&entry{
		source: ioutil.NopCloser(bytes.NewReader([]byte("hello "))),
		cb: func(m *manifest.Stream) {
			calls1++
			require.Equal(t, 1, calls1)
			require.Equal(t, 1, len(m.Ranges))
			require.Equal(t, int64(0), m.Ranges[0].Offset)
			require.Equal(t, int64(6), m.Ranges[0].Length)
		},
	}, &entry{
		source: ioutil.NopCloser(bytes.NewReader([]byte("world!"))),
		cb: func(m *manifest.Stream) {
			calls2++
			require.Equal(t, 1, calls2)
			require.Equal(t, 1, len(m.Ranges))
			require.Equal(t, int64(6), m.Ranges[0].Offset)
			require.Equal(t, int64(6), m.Ranges[0].Length)
		},
	})

	data, err := ioutil.ReadAll(c)
	require.NoError(t, err)
	require.True(t, bytes.Equal(data, []byte("hello world!")))
	c.Cut()
}

func TestConcatTwoCutBefore(t *testing.T) {
	var calls1, calls2 int
	c := newConcat(&entry{
		source: ioutil.NopCloser(bytes.NewReader([]byte("hello "))),
		cb: func(m *manifest.Stream) {
			calls1++
			require.Equal(t, 1, calls1)
			require.Equal(t, 2, len(m.Ranges))
			require.Equal(t, int64(0), m.Ranges[0].Offset)
			require.Equal(t, int64(4), m.Ranges[0].Length)
			require.Equal(t, int64(0), m.Ranges[1].Offset)
			require.Equal(t, int64(2), m.Ranges[1].Length)
			require.NotEqual(t, m.Ranges[0].Blob, m.Ranges[1].Blob)
		},
	}, &entry{
		source: ioutil.NopCloser(bytes.NewReader([]byte("world!"))),
		cb: func(m *manifest.Stream) {
			calls2++
			require.Equal(t, 1, calls2)
			require.Equal(t, 1, len(m.Ranges))
			require.Equal(t, int64(2), m.Ranges[0].Offset)
			require.Equal(t, int64(6), m.Ranges[0].Length)
		},
	})

	var buf [4]byte
	_, err := io.ReadFull(c, buf[:])
	require.NoError(t, err)
	require.True(t, bytes.Equal(buf[:], []byte("hell")))
	c.Cut()

	data, err := ioutil.ReadAll(c)
	require.NoError(t, err)
	require.True(t, bytes.Equal(data, []byte("o world!")))
	c.Cut()
}

func TestConcatTwoCutOn(t *testing.T) {
	var calls1, calls2 int
	c := newConcat(&entry{
		source: ioutil.NopCloser(bytes.NewReader([]byte("hello "))),
		cb: func(m *manifest.Stream) {
			calls1++
			require.Equal(t, 1, calls1)
			require.Equal(t, 1, len(m.Ranges))
			require.Equal(t, int64(0), m.Ranges[0].Offset)
			require.Equal(t, int64(6), m.Ranges[0].Length)
		},
	}, &entry{
		source: ioutil.NopCloser(bytes.NewReader([]byte("world!"))),
		cb: func(m *manifest.Stream) {
			calls2++
			require.Equal(t, 1, calls2)
			require.Equal(t, 1, len(m.Ranges))
			require.Equal(t, int64(0), m.Ranges[0].Offset)
			require.Equal(t, int64(6), m.Ranges[0].Length)
		},
	})

	var buf [6]byte
	_, err := io.ReadFull(c, buf[:])
	require.NoError(t, err)
	require.True(t, bytes.Equal(buf[:], []byte("hello ")))
	c.Cut()

	data, err := ioutil.ReadAll(c)
	require.NoError(t, err)
	require.True(t, bytes.Equal(data, []byte("world!")))
	c.Cut()
}

func TestConcatTwoCutAfter(t *testing.T) {
	var calls1, calls2 int
	c := newConcat(&entry{
		source: ioutil.NopCloser(bytes.NewReader([]byte("hello "))),
		cb: func(m *manifest.Stream) {
			calls1++
			require.Equal(t, 1, calls1)
			require.Equal(t, 1, len(m.Ranges))
			require.Equal(t, int64(0), m.Ranges[0].Offset)
			require.Equal(t, int64(6), m.Ranges[0].Length)
		},
	}, &entry{
		source: ioutil.NopCloser(bytes.NewReader([]byte("world!"))),
		cb: func(m *manifest.Stream) {
			calls2++
			require.Equal(t, 1, calls2)
			require.Equal(t, 2, len(m.Ranges))
			require.Equal(t, int64(6), m.Ranges[0].Offset)
			require.Equal(t, int64(2), m.Ranges[0].Length)
			require.Equal(t, int64(0), m.Ranges[1].Offset)
			require.Equal(t, int64(4), m.Ranges[1].Length)
			require.NotEqual(t, m.Ranges[0].Blob, m.Ranges[1].Blob)
		},
	})

	var buf [8]byte
	_, err := io.ReadFull(c, buf[:])
	require.NoError(t, err)
	require.True(t, bytes.Equal(buf[:], []byte("hello wo")))
	c.Cut()

	data, err := ioutil.ReadAll(c)
	require.NoError(t, err)
	require.True(t, bytes.Equal(data, []byte("rld!")))
	c.Cut()
}
