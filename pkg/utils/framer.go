package utils

import (
	"encoding/binary"
	"io"

	"github.com/zeebo/errs"
)

const frameBytes = 2

var encodeSize = binary.BigEndian.PutUint16
var decodeSize = binary.BigEndian.Uint16

const maxFrameSize = (1 << (frameBytes * 8)) - 1

type FramingReader struct {
	r         io.Reader
	err       error
	frame     []byte
	lastFrame bool
}

func NewFramingReader(r io.Reader) *FramingReader {
	return &FramingReader{
		r:     r,
		frame: make([]byte, 0, maxFrameSize+frameBytes*2),
	}
}

func (f *FramingReader) nextFrame() (err error) {
	if f.lastFrame {
		return io.EOF
	}
	var frameLen int
	dest := f.frame[frameBytes : frameBytes+maxFrameSize]
	for frameLen < maxFrameSize {
		n, err := f.r.Read(dest[frameLen:])
		frameLen += n
		if err != nil {
			if err == io.EOF {
				f.lastFrame = true
				break
			}
			return errs.Wrap(err)
		}
	}
	f.frame = f.frame[:frameBytes+frameLen]
	if f.lastFrame {
		if frameLen > 0 {
			f.frame = append(f.frame, make([]byte, frameBytes)...)
		}
	} else if frameLen != maxFrameSize {
		panic("logic error")
	}
	if frameLen > maxFrameSize || frameLen < 0 {
		panic("logic error")
	}
	encodeSize(f.frame[:frameBytes], uint16(frameLen))
	return nil
}

func (f *FramingReader) Read(p []byte) (n int, err error) {
	if f.err != nil {
		return 0, f.err
	}

	if len(f.frame) == 0 {
		err = f.nextFrame()
		if err != nil {
			f.err = err
			return 0, err
		}
	}

	n = copy(p, f.frame)
	f.frame = f.frame[:copy(f.frame, f.frame[n:])]
	return n, nil
}

type UnframingReader struct {
	r     io.Reader
	err   error
	frame []byte
}

func NewUnframingReader(r io.Reader) *UnframingReader {
	return &UnframingReader{
		r:     r,
		frame: make([]byte, 0, maxFrameSize),
	}
}

func (u *UnframingReader) nextFrame() (err error) {
	var buf [maxFrameSize + frameBytes]byte

	n, err := io.ReadAtLeast(u.r, buf[:], frameBytes)
	if err != nil {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
		return errs.Wrap(err)
	}

	frameSize := int(decodeSize(buf[:frameBytes]))
	if frameSize == 0 {
		return io.EOF
	}
	remaining := frameSize - (n - frameBytes)
	if remaining > 0 {
		_, err = io.ReadFull(u.r, buf[n:n+remaining])
		if err != nil {
			if err == io.EOF {
				err = io.ErrUnexpectedEOF
			}
			return errs.Wrap(err)
		}
	}

	u.frame = u.frame[:frameSize]
	copy(u.frame, buf[frameBytes:frameBytes+frameSize])
	return nil
}

func (u *UnframingReader) Read(p []byte) (n int, err error) {
	if u.err != nil {
		return 0, u.err
	}

	if len(u.frame) == 0 {
		err = u.nextFrame()
		if err != nil {
			u.err = err
			return 0, err
		}
	}

	n = copy(p, u.frame)
	u.frame = u.frame[:copy(u.frame, u.frame[n:])]
	return n, nil
}
