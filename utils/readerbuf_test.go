package utils

import (
	"bytes"
	"crypto/rand"
	"io"
	"io/ioutil"
	"testing"
	"testing/iotest"

	"github.com/stretchr/testify/require"
)

func randData(t *testing.T, amount int) []byte {
	data := make([]byte, amount)
	_, err := rand.Read(data)
	require.NoError(t, err)
	return data
}

func TestReaderBuf(t *testing.T) {
	testReaderBuf(t, 1024*1024, func(r io.Reader) io.Reader { return r })
	testReaderBuf(t, 1024*1024, iotest.DataErrReader)
	testReaderBuf(t, 1024*1024, iotest.HalfReader)
	testReaderBuf(t, 1024*1024, iotest.OneByteReader)
	testReaderBuf(t, 64*1024+3, func(r io.Reader) io.Reader { return r })
	testReaderBuf(t, 64*1024+3, iotest.DataErrReader)
	testReaderBuf(t, 64*1024+3, iotest.HalfReader)
	testReaderBuf(t, 64*1024+3, iotest.OneByteReader)
}

func testReaderBuf(t *testing.T, amount int, wrappers func(io.Reader) io.Reader) {
	expected := randData(t, amount)
	rb := NewReaderBuf(ioutil.NopCloser(wrappers(bytes.NewReader(expected))))
	defer rb.Close()
	actual, err := ioutil.ReadAll(rb)
	require.NoError(t, err)
	require.True(t, bytes.Equal(expected, actual))
}
