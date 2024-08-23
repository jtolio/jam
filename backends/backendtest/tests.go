package backendtest

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"io"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/jtolio/jam/backends"
)

var ctx = context.Background()

type BackendGen func() (b backends.Backend, cleanup func() error, err error)

func listSlice(b backends.Backend, prefix string) (
	rv []string) {
	err := b.List(ctx, prefix, func(ctx context.Context, path string) error {
		rv = append(rv, path)
		return nil
	})
	if err != nil {
		panic(err)
	}
	sort.Strings(rv)
	return rv
}

type suite struct {
	*testing.T
	b backends.Backend
}

func (t *suite) TestGetPutListDelete() {
	data1 := "hello"
	data2 := "hi"
	rv := listSlice(t.b, "")
	require.True(t.T, len(rv) == 0)
	require.NoError(t.T, t.b.Put(ctx, "hello/there", bytes.NewReader([]byte(data1))))
	require.NoError(t.T, t.b.Put(ctx, "hi/there", bytes.NewReader([]byte(data2))))
	rv = listSlice(t.b, "")
	require.True(t.T, len(rv) == 2)
	require.True(t.T, rv[0] == "hello/there")
	require.True(t.T, rv[1] == "hi/there")
	rv = listSlice(t.b, "hello/")
	require.True(t.T, len(rv) == 1)
	require.True(t.T, rv[0] == "hello/there")
	rc, err := t.b.Get(ctx, "hello/there", 0, -1)
	require.NoError(t.T, err)
	data, err := io.ReadAll(io.LimitReader(rc, int64(len([]byte(data1)))))
	require.NoError(t.T, err)
	require.True(t.T, bytes.Equal(data, []byte(data1)))
	require.NoError(t.T, rc.Close())
	require.NoError(t.T, t.b.Delete(ctx, "hello/there"))
	rv = listSlice(t.b, "hello/")
	require.True(t.T, len(rv) == 0)
	rv = listSlice(t.b, "")
	require.True(t.T, len(rv) == 1)
	require.True(t.T, rv[0] == "hi/there")
	rc, err = t.b.Get(ctx, "hi/there", 0, -1)
	require.NoError(t.T, err)
	data, err = io.ReadAll(io.LimitReader(rc, int64(len([]byte(data2)))))
	require.NoError(t.T, err)
	require.True(t.T, bytes.Equal(data, []byte(data2)))
	require.NoError(t.T, rc.Close())
}

func (t *suite) TestOffset() {
	var data [5 * 1024 * 1024]byte
	_, err := rand.Read(data[:])
	require.NoError(t.T, err)
	t.T.Logf("putting 5MB file")
	require.NoError(t.T, t.b.Put(ctx, "testfile", bytes.NewReader(data[:])))
	for i := 0; i < len(data[:]); i += 1024 * 1024 {
		for j := -2; j < 3; j++ {
			if i+j < 0 {
				continue
			}
			for k := 7; k < 10; k++ {
				t.T.Logf("offset %d, length %d", i+j, k)
				rc, err := t.b.Get(ctx, "testfile", int64(i+j), int64(k))
				require.NoError(t.T, err)
				testdata, err := io.ReadAll(io.LimitReader(rc, int64(k)))
				require.NoError(t.T, err)
				require.NoError(t.T, rc.Close())
				require.True(t.T, bytes.Equal(data[i+j:i+j+k], testdata))
			}
		}
	}
}

func (t *suite) TestNonExistent() {
	_, err := t.b.Get(ctx, "non-existing-file", 0, -1)
	require.True(t.T, errors.Is(err, backends.ErrNotExist))
}

type pausingReader struct {
	pre, post []byte
	err       error

	readMtx sync.Mutex
	waitMtx sync.Mutex
}

func newPausingReader(pre, post []byte, err error) *pausingReader {
	pr := &pausingReader{
		pre:  pre,
		post: post,
		err:  err,
	}
	pr.waitMtx.Lock()
	return pr
}

func (pr *pausingReader) Read(p []byte) (n int, err error) {
	pr.readMtx.Lock()
	pr.readMtx.Unlock()
	if len(pr.pre) > 0 {
		n = copy(p, pr.pre)
		pr.pre = pr.pre[n:]
		return n, nil
	}
	if len(pr.post) > 0 {
		pr.pre = pr.post
		pr.post = nil
		pr.waitMtx.Unlock()
		pr.readMtx.Lock()
		return pr.Read(p)
	}
	if pr.err != nil {
		return 0, pr.err
	}
	return 0, io.EOF
}

func (pr *pausingReader) Unpause() {
	pr.readMtx.Unlock()
}

func (pr *pausingReader) Wait() {
	pr.waitMtx.Lock()
	pr.waitMtx.Unlock()
}

func (t *suite) TestPartialPut_Success() {
	require.True(t.T, len(listSlice(t.b, "")) == 0)
	require.NoError(t.T, t.b.Put(ctx, "hello/there", bytes.NewReader([]byte("hello"))))

	var data [2 * 1024 * 1024]byte
	_, err := rand.Read(data[:])
	require.NoError(t.T, err)

	pr := newPausingReader(data[:1024*1024], data[1024*1024:], nil)

	errch := make(chan error, 1)
	go func() {
		errch <- t.b.Put(ctx, "partial/put", pr)
	}()

	pr.Wait()

	// make sure partial/put doesn't exist yet
	rv := listSlice(t.b, "")
	require.True(t.T, len(rv) == 1)
	require.True(t.T, rv[0] == "hello/there")

	_, err = t.b.Get(ctx, "partial/put", 0, -1)
	require.True(t.T, errors.Is(err, backends.ErrNotExist))

	pr.Unpause()

	require.NoError(t.T, <-errch)

	rv = listSlice(t.b, "")
	require.True(t.T, len(rv) == 2)
	require.True(t.T, rv[0] == "hello/there")
	require.True(t.T, rv[1] == "partial/put")

	rc, err := t.b.Get(ctx, "partial/put", 0, -1)
	require.NoError(t.T, err)
	require.NoError(t.T, rc.Close())
}

func (t *suite) TestPartialPut_Failure() {
	require.True(t.T, len(listSlice(t.b, "")) == 0)
	require.NoError(t.T, t.b.Put(ctx, "hello/there", bytes.NewReader([]byte("hello"))))

	var data [2 * 1024 * 1024]byte
	_, err := rand.Read(data[:])
	require.NoError(t.T, err)

	var myError = errors.New("test error")

	pr := newPausingReader(data[:1024*1024], data[1024*1024:], myError)

	errch := make(chan error, 1)
	go func() {
		errch <- t.b.Put(ctx, "partial/put", pr)
	}()

	pr.Wait()

	// make sure partial/put doesn't exist yet
	rv := listSlice(t.b, "")
	require.True(t.T, len(rv) == 1)
	require.True(t.T, rv[0] == "hello/there")

	_, err = t.b.Get(ctx, "partial/put", 0, -1)
	require.True(t.T, errors.Is(err, backends.ErrNotExist))

	pr.Unpause()

	require.True(t.T, errors.Is(<-errch, myError))

	rv = listSlice(t.b, "")
	require.True(t.T, len(rv) == 1)
	require.True(t.T, rv[0] == "hello/there")

	_, err = t.b.Get(ctx, "partial/put", 0, -1)
	require.True(t.T, errors.Is(err, backends.ErrNotExist))
}

func (t *suite) TestPutOverwrite() {
	data1 := "testing testing testing"
	require.True(t.T, len(listSlice(t.b, "")) == 0)
	for i := 0; i < 2; i++ {
		require.NoError(t.T, t.b.Put(ctx, "hello/there", bytes.NewReader([]byte(data1))))
		rv := listSlice(t.b, "")
		require.True(t.T, len(rv) == 1)
		require.True(t.T, rv[0] == "hello/there")
		rc, err := t.b.Get(ctx, "hello/there", 0, -1)
		require.NoError(t.T, err)
		data, err := io.ReadAll(io.LimitReader(rc, int64(len([]byte(data1)))))
		require.NoError(t.T, err)
		require.True(t.T, bytes.Equal(data, []byte(data1)))
		require.NoError(t.T, rc.Close())
	}
}

func (t *suite) TestDeleteMissing() {
	err := t.b.Delete(ctx, "non-existing-file")
	require.NoError(t.T, err)
}

func (t *suite) TestLength() {
	t.Skip()
	// TODO: this needs to only confirm that at least the length requested is
	// indeed returned (it's okay if more than the length requested is
	// returned)
}

func (t *suite) TestHierarchy() {
	var data [1024]byte
	_, err := rand.Read(data[:])
	require.NoError(t.T, err)
	require.NoError(t.T, t.b.Put(ctx, "a/b/c/d/e/f", bytes.NewReader(data[:])))
	require.NoError(t.T, t.b.Put(ctx, "a/b/c/d/e/g", bytes.NewReader(data[:])))
	require.NoError(t.T, t.b.Put(ctx, "a/b/c/d/e/h", bytes.NewReader(data[:])))
	require.NoError(t.T, t.b.Put(ctx, "a/b/c/d/i", bytes.NewReader(data[:])))
	require.NoError(t.T, t.b.Put(ctx, "b/c/d/i", bytes.NewReader(data[:])))
	require.Equal(t.T, listSlice(t.b, ""), []string{
		"a/b/c/d/e/f",
		"a/b/c/d/e/g",
		"a/b/c/d/e/h",
		"a/b/c/d/i",
		"b/c/d/i",
	})
	require.Equal(t.T, listSlice(t.b, "a/"), []string{
		"a/b/c/d/e/f",
		"a/b/c/d/e/g",
		"a/b/c/d/e/h",
		"a/b/c/d/i",
	})
	require.Equal(t.T, listSlice(t.b, "b/"), []string{
		"b/c/d/i",
	})
	require.Equal(t.T, listSlice(t.b, "a/b/c/d/e/"), []string{
		"a/b/c/d/e/f",
		"a/b/c/d/e/g",
		"a/b/c/d/e/h",
	})
	require.Equal(t.T, listSlice(t.b, "a/b/c/d/"), []string{
		"a/b/c/d/e/f",
		"a/b/c/d/e/g",
		"a/b/c/d/e/h",
		"a/b/c/d/i",
	})
	require.Equal(t.T, listSlice(t.b, "a/b/c/d/i/"), []string(nil))
}

func RunSuite(t *testing.T, gen BackendGen) {
	st := reflect.TypeOf(&suite{})
	for i := 0; i < st.NumMethod(); i++ {
		m := st.Method(i)
		if !strings.HasPrefix(m.Name, "Test") {
			continue
		}
		b, cleanup, err := gen()
		if err != nil {
			t.Fatal(err)
		}
		t.Run(m.Name, func(t *testing.T) {
			m.Func.Call([]reflect.Value{reflect.ValueOf(&suite{
				T: t,
				b: b,
			})})
		})
		err = b.Close()
		if err != nil {
			t.Fatal(err)
		}
		if cleanup != nil {
			err = cleanup()
			if err != nil {
				t.Fatal(err)
			}
		}
	}
}
