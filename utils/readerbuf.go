package utils

import (
	"bytes"
	"io"
	"sync"

	"github.com/zeebo/errs"
)

const bufSize = 32 * 1024

// ReaderBuf wraps a Reader with a goroutine that fills an internal
// buffer of the next read. Using ReaderBuf will keep the next read
// pipelined and ready. Calling a read (however small) will trigger
// the next pipelined read up to the internal buffer size. If you want
// many small reads, this should probably be wrapped in a bufio.Reader.
// ReaderBuf should be closed when done to make sure the goroutine
// gets shut down.
type ReaderBuf struct {
	r     io.ReadCloser
	mtx   sync.Mutex
	fill  *sync.Cond
	drain *sync.Cond
	buf   bytes.Buffer
	err   error
}

func NewReaderBuf(r io.ReadCloser) *ReaderBuf {
	rb := &ReaderBuf{
		r: r,
	}
	rb.fill = sync.NewCond(&rb.mtx)
	rb.drain = sync.NewCond(&rb.mtx)
	go rb.run()
	return rb
}

func (rb *ReaderBuf) run() {
	var p [bufSize]byte
	rb.mtx.Lock()
	for {
		for rb.buf.Len() >= bufSize && rb.r != nil {
			rb.drain.Wait()
		}
		if rb.r == nil {
			rb.mtx.Unlock()
			return
		}
		amount := bufSize - rb.buf.Len()
		r := rb.r
		rb.mtx.Unlock()
		n, err := r.Read(p[:amount])
		rb.mtx.Lock()
		must(rb.buf.Write(p[:n]))
		rb.fill.Signal()
		if err != nil {
			rb.err = err
			rb.mtx.Unlock()
			return
		}
	}
}

func must(n int, err error) {
	if err != nil {
		panic(err)
	}
}

// Read will block until there is data in the internal buffer, but
// will otherwise read out of the internal buffer and signal that
// the next read can begin.
func (rb *ReaderBuf) Read(p []byte) (n int, err error) {
	rb.mtx.Lock()
	defer rb.mtx.Unlock()
	for rb.err == nil && rb.buf.Len() == 0 && rb.r != nil {
		rb.fill.Wait()
	}
	if rb.buf.Len() > 0 {
		if len(p) > rb.buf.Len() {
			p = p[:rb.buf.Len()]
		}
		n, err = rb.buf.Read(p)
		rb.drain.Signal()
		if err != nil {
			panic(err)
		}
		if n <= 0 {
			panic("expected bytes")
		}
		return n, nil
	}
	if rb.r == nil {
		return 0, errs.New("closed")
	}
	if rb.err != nil {
		return 0, rb.err
	}
	panic("unreachable")
}

// Close shuts down any outstanding reads and the wrapped ReadCloser.
func (rb *ReaderBuf) Close() error {
	rb.mtx.Lock()
	r := rb.r
	rb.r = nil
	rb.fill.Signal()
	rb.drain.Signal()
	rb.mtx.Unlock()
	return r.Close()
}
