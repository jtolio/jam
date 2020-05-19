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
}

func newHashConfirmReader(r io.ReadCloser, h hash.Hash, expected []byte) *hashConfirmReader {
	return &hashConfirmReader{r: r, h: h, expected: expected}
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
		if !bytes.Equal(hcr.h.Sum(nil), hcr.expected) {
			err = errs.New("file changed while reading")
		}
	}
	return n, err
}

func (hcr *hashConfirmReader) Close() error {
	return hcr.r.Close()
}
