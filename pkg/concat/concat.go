package concat

import (
	"context"
	"io"

	"github.com/jtolds/jam/pkg/manifest"
)

type Concatenator struct {
}

func NewConcatenator() *Concatenator { return &Concatenator{} }

func (c *Concatenator) Add(ctx context.Context, r io.Reader) (result *manifest.Stream, err error) {
	panic("TODO")
}

func (c *Concatenator) Commit(ctx context.Context) error {
	panic("TODO")
}

func (c *Concatenator) Destination(ctx context.Context, length int64) (*Destination, error) {
	panic("TODO")
}

func (c *Concatenator) Close() error {
	panic("TODO")
}

type Destination struct {
}

func (d *Destination) Read(p []byte) (n int, err error) {
	panic("TODO")
}

func (d *Destination) Commit(blob string) error {
	panic("TODO")
}

func (d *Destination) Abort() error {
	panic("TODO")
}
