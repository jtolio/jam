package enc

import (
	"bufio"
	"io"
)

type encodedReader struct {
	r        io.Reader
	c        Codec
	key      *[32]byte
	blockNum int64
	inbuf    []byte
	outbuf   []byte
}

// EncodeReader applies a Codec's encoding to a given reader, starting at the given
// startingBlockNum (usually 0).
func EncodeReader(r io.Reader, c Codec, key *[32]byte, startingBlockNum int64) io.Reader {
	return &encodedReader{
		r:        bufio.NewReader(r),
		c:        c,
		key:      key,
		blockNum: startingBlockNum,
		inbuf:    make([]byte, c.DecodedBlockSize()),
		outbuf:   make([]byte, 0, c.EncodedBlockSize()),
	}
}

func (r *encodedReader) read(p []byte) (n int, err error) {
	if len(r.outbuf) <= 0 {
		_, err = io.ReadFull(r.r, r.inbuf)
		if err != nil {
			return 0, err
		}
		r.outbuf, err = r.c.Encode(r.outbuf, r.inbuf, r.key, r.blockNum)
		if err != nil {
			return 0, err
		}
		r.blockNum++
	}

	n = copy(p, r.outbuf)
	copy(r.outbuf, r.outbuf[n:])
	r.outbuf = r.outbuf[:len(r.outbuf)-n]
	return n, nil
}

func (r *encodedReader) Read(p []byte) (n int, err error) {
	for {
		b, err := r.read(p)
		n, p = n+b, p[b:]
		if err != nil || len(p) == 0 {
			return n, err
		}
	}
}

// DecodeReader applies a Codec's decoding to a given reader, starting at the given
// startingBlockNum (usually 0).
func DecodeReader(r io.Reader, c Codec, key *[32]byte, startingBlockNum int64) io.Reader {
	return EncodeReader(r, Reverse(c), key, startingBlockNum)
}
