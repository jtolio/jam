package pathdb

import (
	"context"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/jtolds/jam/pkg/manifest"
)

var (
	ctx = context.Background()
)

type listFn func(ctx context.Context, prefix, delimiter string,
	cb func(ctx context.Context, path string, content *manifest.Content) error) error

func collectPaths(list listFn, prefix, delimiter string) (rv []string) {
	err := list(ctx, prefix, delimiter,
		func(ctx context.Context, path string, content *manifest.Content) error {
			if content == nil {
				rv = append(rv, "PRE "+path)
			} else {
				rv = append(rv, "OBJ "+path)
			}
			return nil
		})
	if err != nil {
		panic(err)
	}
	return rv
}

func TestPathDB(t *testing.T) {
	db := New(nil, nil)
	paths := []string{
		"", "a", "b", "/", "/a", "/b", "a/", "b/", "a/a", "a/b", "b/a", "b/b",
		"a/a/a", "a/a/b", "a/b/a", "a/b/b", "b/a/a", "b/a/b", "b/b/a", "b/b/b",
	}
	var allPaths []string
	for _, path := range paths {
		db.Put(ctx, path, &manifest.Content{})
		allPaths = append(allPaths, "OBJ "+path)
	}
	sort.Strings(allPaths)
	require.Equal(t, allPaths, collectPaths(db.List, "", ""))
	require.Equal(t, []string{"OBJ a", "OBJ a/", "OBJ a/a", "OBJ a/a/a", "OBJ a/a/b",
		"OBJ a/b", "OBJ a/b/a", "OBJ a/b/b"},
		collectPaths(db.List, "a", ""))
	require.Equal(t, []string{"OBJ a/", "OBJ a/a", "OBJ a/a/a", "OBJ a/a/b", "OBJ a/b",
		"OBJ a/b/a", "OBJ a/b/b"},
		collectPaths(db.List, "a/", ""))
	require.Equal(t, []string{"OBJ a/", "OBJ a/a", "PRE a/a", "OBJ a/b", "PRE a/b"},
		collectPaths(db.List, "a/", "/"))
}
