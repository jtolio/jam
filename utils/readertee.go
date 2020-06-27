package utils

import (
	"io"
	"sync"
)

// ReaderTee turns a single Reader into two Readers that proceed in
// lockstep together with the same data.
func ReaderTee(r io.Reader) (io.Reader, io.Reader) {
	wrapper := &readerTee{
		r:  r,
		cv: sync.NewCond(new(sync.Mutex)),
	}
	return ReaderFunc(wrapper.Read1), ReaderFunc(wrapper.Read2)
}

type ReaderFunc func(p []byte) (n int, err error)

func (f ReaderFunc) Read(p []byte) (n int, err error) { return f(p) }

type readerTee struct {
	r      io.Reader
	cv     *sync.Cond
	offset [2]int
	buflen int
	buf    [32 * 1024]byte
	err    error
}

func (t *readerTee) Read1(p []byte) (n int, err error) {
	return t.read(p, 0)
}

func (t *readerTee) Read2(p []byte) (n int, err error) {
	return t.read(p, 1)
}

func (t *readerTee) read(p []byte, reader int) (n int, err error) {
	t.cv.L.Lock()
	defer t.cv.L.Unlock()

	if t.buflen == 0 {
		// init case
		t.fill()
	}

	for {
		// does this reader have data? return it
		if t.offset[reader] < t.buflen {
			n = copy(p, t.buf[t.offset[reader]:t.buflen])
			t.offset[reader] += n
			return n, nil
		}

		// is there an error? return it
		if t.err != nil {
			return 0, t.err
		}

		// do other readers have data? wait and start over
		if t.offset[1-reader] < t.buflen {
			t.cv.Wait()
			continue
		}

		t.fill()
	}
}

func (t *readerTee) fill() {
	t.offset = [2]int{0, 0}
	for {
		n, err := t.r.Read(t.buf[:])
		if err != nil || n > 0 {
			t.buflen = n
			t.err = err
			t.cv.Broadcast()
			return
		}
	}
}
