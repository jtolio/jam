package utils

import (
	"github.com/zeebo/errs"
)

func Parallel(fn ...func() error) error {
	errch := make(chan error, len(fn))
	for _, f := range fn {
		go func(f func() error) {
			errch <- f()
		}(f)
	}
	var eg errs.Group
	for range fn {
		eg.Add(<-errch)
	}
	return eg.Err()
}
