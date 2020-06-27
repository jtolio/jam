package utils

import (
	"bytes"
	"io"
	"io/ioutil"
	"testing"
	"testing/iotest"

	"github.com/stretchr/testify/require"
)

func TestReaderCompareHappy(t *testing.T) {
	testReaderCompareHappy(t, 1024*1024, func(r io.Reader) io.Reader { return r })
	testReaderCompareHappy(t, 1024*1024, iotest.DataErrReader)
	testReaderCompareHappy(t, 1024*1024, iotest.HalfReader)
	testReaderCompareHappy(t, 1024*1024, iotest.OneByteReader)
	testReaderCompareHappy(t, 64*1024+3, func(r io.Reader) io.Reader { return r })
	testReaderCompareHappy(t, 64*1024+3, iotest.DataErrReader)
	testReaderCompareHappy(t, 64*1024+3, iotest.HalfReader)
	testReaderCompareHappy(t, 64*1024+3, iotest.OneByteReader)
}

func testReaderCompareHappy(t *testing.T, amount int, wrappers func(io.Reader) io.Reader) {
	expected := randData(t, amount)

	rc := ReaderCompare(ioutil.NopCloser(bytes.NewReader(expected)),
		ioutil.NopCloser(wrappers(bytes.NewReader(expected))))
	defer rc.Close()
	actual, err := ioutil.ReadAll(rc)
	require.NoError(t, err)
	require.True(t, bytes.Equal(expected, actual))
}

func TestReaderCompareDifferentLengths(t *testing.T) {
	expected := randData(t, 1024*1024)

	rc := ReaderCompare(ioutil.NopCloser(bytes.NewReader(expected)),
		ioutil.NopCloser(bytes.NewReader(expected[:512*1024])))
	defer rc.Close()
	_, err := ioutil.ReadAll(rc)
	require.True(t, ErrComparisonMismatch.Has(err))
}

func TestReaderCompareEarlyError(t *testing.T) {
	expected := randData(t, 1024*1024)

	rc := ReaderCompare(ioutil.NopCloser(bytes.NewReader(expected)),
		ioutil.NopCloser(iotest.TimeoutReader(bytes.NewReader(expected))))
	defer rc.Close()
	_, err := ioutil.ReadAll(rc)
	require.True(t, ErrComparisonMismatch.Has(err))
}

func TestReaderCompareDifferentBytes(t *testing.T) {
	testReaderCompareDifferentBytes(t, false)
	testReaderCompareDifferentBytes(t, true)
}

func testReaderCompareDifferentBytes(t *testing.T, differ bool) {
	data1 := randData(t, 1024*1024)
	data2 := make([]byte, len(data1))
	copy(data2, data1[:512*1024])
	if differ {
		copy(data2[512*1024:], randData(t, 512*1024))
	} else {
		copy(data2[512*1024:], data1[512*1024:])
	}

	rc := ReaderCompare(ioutil.NopCloser(bytes.NewReader(data1)),
		ioutil.NopCloser(bytes.NewReader(data2)))
	defer rc.Close()
	actual, err := ioutil.ReadAll(rc)
	if differ {
		require.True(t, ErrComparisonMismatch.Has(err))
	} else {
		require.NoError(t, err)
		require.True(t, bytes.Equal(data1, actual))
	}
}
