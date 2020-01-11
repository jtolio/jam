package backendtest

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"reflect"
	"sort"
	"strings"
	"testing"

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

func ane(err error) {
	if err != nil {
		panic(fmt.Sprintf("%+v", err))
	}
}

func assert(val bool) {
	if !val {
		panic("assertion failed")
	}
}

type suite struct {
	*testing.T
	b backends.Backend
}

func (t *suite) TestGetPutListDelete() {
	rv := listSlice(t.b, "")
	assert(len(rv) == 0)
	ane(t.b.Put(ctx, "hello/there", bytes.NewReader([]byte("hello"))))
	ane(t.b.Put(ctx, "hi/there", bytes.NewReader([]byte("hi"))))
	rv = listSlice(t.b, "")
	assert(len(rv) == 2)
	assert(rv[0] == "hello/there")
	assert(rv[1] == "hi/there")
	rv = listSlice(t.b, "hello/")
	assert(len(rv) == 1)
	assert(rv[0] == "hello/there")
	rc, err := t.b.Get(ctx, "hello/there", 0)
	ane(err)
	data, err := ioutil.ReadAll(rc)
	ane(err)
	assert(bytes.Equal(data, []byte("hello")))
	ane(rc.Close())
	ane(t.b.Delete(ctx, "hello/there"))
	rv = listSlice(t.b, "hello/")
	assert(len(rv) == 0)
	rv = listSlice(t.b, "")
	assert(len(rv) == 1)
	assert(rv[0] == "hi/there")
	rc, err = t.b.Get(ctx, "hi/there", 0)
	ane(err)
	data, err = ioutil.ReadAll(rc)
	ane(err)
	assert(bytes.Equal(data, []byte("hi")))
	ane(rc.Close())
}

func (t *suite) TestOffset() {
	var data [10 * 1024]byte
	_, err := rand.Read(data[:])
	ane(err)
	ane(t.b.Put(ctx, "testfile", bytes.NewReader(data[:])))
	for i := 0; i <= len(data[:]); i++ {
		rc, err := t.b.Get(ctx, "testfile", int64(i))
		ane(err)
		testdata, err := ioutil.ReadAll(rc)
		ane(err)
		ane(rc.Close())
		assert(bytes.Equal(data[i:], testdata))
	}
}

func assertSlicesEqual(a, b []string) {
	assert(len(a) == len(b))
	for i, v := range a {
		assert(v == b[i])
	}
}

func (t *suite) TestHierarchy() {
	var data [1024]byte
	_, err := rand.Read(data[:])
	ane(err)
	ane(t.b.Put(ctx, "a/b/c/d/e/f", bytes.NewReader(data[:])))
	ane(t.b.Put(ctx, "a/b/c/d/e/g", bytes.NewReader(data[:])))
	ane(t.b.Put(ctx, "a/b/c/d/e/h", bytes.NewReader(data[:])))
	ane(t.b.Put(ctx, "a/b/c/d/i", bytes.NewReader(data[:])))
	ane(t.b.Put(ctx, "b/c/d/i", bytes.NewReader(data[:])))
	assertSlicesEqual(listSlice(t.b, ""), []string{
		"a/b/c/d/e/f",
		"a/b/c/d/e/g",
		"a/b/c/d/e/h",
		"a/b/c/d/i",
		"b/c/d/i",
	})
	assertSlicesEqual(listSlice(t.b, "a/"), []string{
		"a/b/c/d/e/f",
		"a/b/c/d/e/g",
		"a/b/c/d/e/h",
		"a/b/c/d/i",
	})
	assertSlicesEqual(listSlice(t.b, "b/"), []string{
		"b/c/d/i",
	})
	assertSlicesEqual(listSlice(t.b, "a/b/c/d/e/"), []string{
		"a/b/c/d/e/f",
		"a/b/c/d/e/g",
		"a/b/c/d/e/h",
	})
	assertSlicesEqual(listSlice(t.b, "a/b/c/d/"), []string{
		"a/b/c/d/e/f",
		"a/b/c/d/e/g",
		"a/b/c/d/e/h",
		"a/b/c/d/i",
	})
	assertSlicesEqual(listSlice(t.b, "a/b/c/d/i/"), []string{})
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
