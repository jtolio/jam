package session

import (
	"bytes"
	"hash"
	"io"

	"github.com/zeebo/errs"
)

type hashConfirmReader struct {
	r        io.ReadCloser
	h        hash.Hash
	expected []byte
	message  string
}

func newHashConfirmReader(r io.ReadCloser, h hash.Hash, expected []byte, message string) *hashConfirmReader {
	return &hashConfirmReader{r: r, h: h, expected: expected, message: message}
}

func (hcr *hashConfirmReader) Read(p []byte) (n int, err error) {
	n, err = hcr.r.Read(p)
	if n > 0 {
		_, writeErr := hcr.h.Write(p[:n])
		if writeErr != nil {
			return n, errs.Combine(writeErr, err)
		}
	}
	if err == io.EOF {
		actual := hcr.h.Sum(nil)
		if !bytes.Equal(actual, hcr.expected) {
			err = errs.New("%s", hcr.message)
		}
	}
	return n, err
}

func (hcr *hashConfirmReader) Close() error {
	return hcr.r.Close()
}
