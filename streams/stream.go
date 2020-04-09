package streams

import (
	"context"
	"fmt"
	"io"

	"github.com/jtolds/jam/backends"
	"github.com/jtolds/jam/manifest"
)

type Stream struct {
	backend       backends.Backend
	stream        *manifest.Stream
	currentRange  io.ReadCloser
	currentOffset int64
	length        int64
	ctx           context.Context
}

var _ io.ReadCloser = (*Stream)(nil)
var _ io.Seeker = (*Stream)(nil)

// Open returns a Stream ready for reading. Close only needs to be called if
func Open(ctx context.Context, backend backends.Backend, stream *manifest.Stream) (*Stream, error) {
	var length int64
	for _, r := range stream.Ranges {
		length += r.Length
	}
	return &Stream{
		backend: backend,
		stream:  stream,
		length:  length,
		ctx:     ctx,
	}, nil
}

// Fork is safe to do on a closed or open stream and returns a new valid
// stream at the same offset
func (f *Stream) Fork(ctx context.Context) *Stream {
	return &Stream{
		backend:       f.backend,
		stream:        f.stream,
		currentOffset: f.currentOffset,
		length:        f.length,
		ctx:           ctx,
	}
}

func (f *Stream) Read(p []byte) (n int, err error) {
	if f.currentRange == nil {
		err = f.open()
		if err != nil {
			return 0, err
		}
	}
	n, err = f.currentRange.Read(p)
	f.currentOffset += int64(n)
	if err == io.EOF {
		err = f.Close()
	}
	return n, err
}

func (f *Stream) open() error {
	offset := f.currentOffset
	for _, r := range f.stream.Ranges {
		if offset-r.Length < 0 {
			currentRange, err := openRange(f.ctx, f.backend, r, offset)
			if err != nil {
				return err
			}
			f.currentRange = currentRange
			return nil
		}
		offset -= r.Length
	}
	return io.EOF
}

func (f *Stream) Close() error {
	currentRange := f.currentRange
	f.currentRange = nil
	if currentRange != nil {
		return currentRange.Close()
	}
	return nil
}

func (f *Stream) Seek(offset int64, whence int) (int64, error) {
	oldOffset := f.currentOffset
	switch whence {
	case io.SeekStart:
		f.currentOffset = offset
	case io.SeekCurrent:
		f.currentOffset += offset
	case io.SeekEnd:
		f.currentOffset = f.length + offset
	default:
		return f.currentOffset, fmt.Errorf("invalid whence")
	}
	if oldOffset != f.currentOffset {
		err := f.Close()
		if err != nil {
			return f.currentOffset, err
		}
	}
	return f.currentOffset, nil
}

func (f *Stream) Length() int64 { return f.length }
