package utils

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"sync"

	"github.com/zeebo/errs"
)

var (
	ErrComparisonMismatch = errs.Class("reader comparer mismatch")
)

type readerComparer struct {
	readers []io.ReadCloser
	amounts []int
	errs    []error
	buffers [][]byte
}

// ReaderCompare will return an io.ReadCloser that will error if
// any Reader returns data that doesn't match the others.
func ReaderCompare(readers ...io.ReadCloser) io.ReadCloser {
	if len(readers) <= 0 {
		return ioutil.NopCloser(bytes.NewReader(nil))
	}
	if len(readers) == 1 {
		return readers[0]
	}
	buffered := make([]io.ReadCloser, 0, len(readers))
	for _, r := range readers {
		buffered = append(buffered, NewReaderBuf(r))
	}
	return &readerComparer{
		readers: buffered,
		amounts: make([]int, len(buffered)),
		errs:    make([]error, len(buffered)),
		buffers: make([][]byte, len(buffered)),
	}
}

// readFull is like io.ReadFull but doesn't turn io.EOF into io.ErrUnexpectedEOF
func readFull(r io.Reader, buf []byte) (n int, err error) {
	for n < len(buf) && err == nil {
		var nn int
		nn, err = r.Read(buf[n:])
		n += nn
	}
	if n == len(buf) {
		err = nil
	}
	return
}

func (rc *readerComparer) Read(p []byte) (n int, err error) {
	if len(p) <= 0 {
		return 0, nil
	}

	for i := 1; i < len(rc.readers); i++ {
		if len(rc.buffers[i]) < len(p) {
			rc.buffers[i] = make([]byte, len(p))
		}
	}

	rc.buffers[0] = p
	var wg sync.WaitGroup
	wg.Add(len(rc.readers))

	for i := range rc.readers {
		go func(i int) {
			defer wg.Done()
			rc.amounts[i], rc.errs[i] = readFull(rc.readers[i], rc.buffers[i])
		}(i)
	}

	wg.Wait()

	for i := 1; i < len(rc.readers); i++ {
		if rc.errs[0] != rc.errs[i] {
			return 0, ErrComparisonMismatch.New("errors mismatch %q != %q",
				fmt.Sprintf("%+v", rc.errs[0]), fmt.Sprintf("%+v", rc.errs[i]))
		}
		if rc.amounts[0] != rc.amounts[i] {
			return 0, ErrComparisonMismatch.New("lengths mismatch: %d != %d",
				rc.amounts[0], rc.amounts[i])
		}
		if !bytes.Equal(rc.buffers[0][:rc.amounts[0]], rc.buffers[i][:rc.amounts[i]]) {
			return 0, ErrComparisonMismatch.New("bytes mismatch")
		}
	}

	return rc.amounts[0], rc.errs[0]
}

func (rc *readerComparer) Close() error {
	fns := make([]func() error, 0, len(rc.readers))
	for _, r := range rc.readers {
		fns = append(fns, r.Close)
	}
	return Parallel(fns...)
}
