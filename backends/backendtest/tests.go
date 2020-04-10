package backendtest

import (
	"bytes"
	"context"
	"crypto/rand"
	"io"
	"io/ioutil"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/jtolds/jam/backends"
)

var ctx = context.Background()

type BackendGen func() (b backends.Backend, closer func() error, err error)

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
	rc, err := t.b.Get(ctx, "hello/there", 0)
	require.NoError(t.T, err)
	data, err := ioutil.ReadAll(io.LimitReader(rc, int64(len([]byte(data1)))))
	require.NoError(t.T, err)
	require.True(t.T, bytes.Equal(data, []byte(data1)))
	require.NoError(t.T, rc.Close())
	require.NoError(t.T, t.b.Delete(ctx, "hello/there"))
	rv = listSlice(t.b, "hello/")
	require.True(t.T, len(rv) == 0)
	rv = listSlice(t.b, "")
	require.True(t.T, len(rv) == 1)
	require.True(t.T, rv[0] == "hi/there")
	rc, err = t.b.Get(ctx, "hi/there", 0)
	require.NoError(t.T, err)
	data, err = ioutil.ReadAll(io.LimitReader(rc, int64(len([]byte(data2)))))
	require.NoError(t.T, err)
	require.True(t.T, bytes.Equal(data, []byte(data2)))
	require.NoError(t.T, rc.Close())
}

func (t *suite) TestOffset() {
	var data [10 * 1024]byte
	_, err := rand.Read(data[:])
	require.NoError(t.T, err)
	require.NoError(t.T, t.b.Put(ctx, "testfile", bytes.NewReader(data[:])))
	for i := 0; i <= len(data[:]); i++ {
		rc, err := t.b.Get(ctx, "testfile", int64(i))
		require.NoError(t.T, err)
		testdata, err := ioutil.ReadAll(io.LimitReader(rc, int64(len(data)-i)))
		require.NoError(t.T, err)
		require.NoError(t.T, rc.Close())
		require.True(t.T, bytes.Equal(data[i:], testdata))
	}
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
		b, closer, err := gen()
		if err != nil {
			t.Fatal(err)
		}
		t.Run(m.Name, func(t *testing.T) {
			m.Func.Call([]reflect.Value{reflect.ValueOf(&suite{
				T: t,
				b: b,
			})})
		})
		err = closer()
		if err != nil {
			t.Fatal(err)
		}
	}
}
