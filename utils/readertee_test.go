package utils

import (
	"bytes"
	"crypto/rand"
	"io"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

type WriterFunc func(p []byte) (n int, err error)

func (f WriterFunc) Write(p []byte) (n int, err error) { return f(p) }

func testReaderTee(t *testing.T, buffer1, buffer2 int) {
	var data [128 * 1024]byte
	_, err := rand.Read(data[:])
	require.NoError(t, err)

	r1, r2 := ReaderTee(bytes.NewReader(data[:]))

	var w1, w2 bytes.Buffer
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, err := io.CopyBuffer(WriterFunc(w1.Write), r1, make([]byte, buffer1))
		require.NoError(t, err)
	}()

	go func() {
		defer wg.Done()
		_, err := io.CopyBuffer(WriterFunc(w2.Write), r2, make([]byte, buffer2))
		require.NoError(t, err)
	}()

	wg.Wait()

	require.True(t, bytes.Equal(w1.Bytes(), data[:]))
	require.True(t, bytes.Equal(w2.Bytes(), data[:]))
}

func TestReaderTee(t *testing.T) {
	testReaderTee(t, 32*1024, 32*1024)
	testReaderTee(t, 32*1024-1, 32*1024+1)
	testReaderTee(t, 47, 16*1024)
	testReaderTee(t, 64*1024, 1024)
	testReaderTee(t, 128*1024, 4096)
}
