package utils

import (
	"context"
	"sort"
)

type Lister interface {
	List(ctx context.Context, prefix string, cb func(ctx context.Context, path string) error) error
}

func SortedList(ctx context.Context, backend Lister, prefix string,
	cb func(ctx context.Context, path string) error) error {

	var paths []string
	err := backend.List(ctx, prefix,
		func(ctx context.Context, path string) error {
			paths = append(paths, path)
			return nil
		})
	if err != nil {
		return err
	}
	sort.Strings(paths)
	for _, path := range paths {
		err = cb(ctx, path)
		if err != nil {
			return err
		}
	}
	return nil
}
