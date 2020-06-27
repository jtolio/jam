package utils

import (
	"bytes"
	"io"
	"io/ioutil"

	"github.com/zeebo/errs"
)

var (
	ErrComparisonMismatch = errs.Class("reader comparer mismatch")
)

type readerComparer struct {
	readers []io.ReadCloser
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
	return &readerComparer{readers: buffered}
}

func (rc *readerComparer) Read(p []byte) (n int, err error) {
	if len(p) <= 0 {
		return 0, nil
	}
	n1, err1 := rc.readers[0].Read(p)
	for _, o := range rc.readers[1:] {
		var p2 []byte
		if n1 <= 0 {
			p2 = make([]byte, len(p))
		} else {
			p2 = make([]byte, n1)
		}
		n2, err2 := io.ReadFull(o, p2)
		if n1 != n2 {
			return n1, ErrComparisonMismatch.New("lengths mismatch: %d != %d", n1, n2)
		}
		if !bytes.Equal(p[:n1], p2[:n2]) {
			return n1, ErrComparisonMismatch.New("bytes mismatch")
		}
		if err1 != err2 {
			return n1, ErrComparisonMismatch.New("errors mismatch on %d/%d bytes: %q != %q", n1, len(p), err1, err2)
		}
	}
	return n1, err1
}

func (rc *readerComparer) Close() error {
	fns := make([]func() error, 0, len(rc.readers))
	for _, r := range rc.readers {
		fns = append(fns, r.Close)
	}
	return Parallel(fns...)
}
