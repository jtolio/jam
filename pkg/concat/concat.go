package session

import (
	"context"
	"io"
)

type Concatenator struct {
}

func NewConcatenator() *Concatenator { return &Concatenator{} }

func (c *Concatenator) Add(ctx context.Context, r io.Reader) (name string, err error) {
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

func (d *Destination) Commit(name string) error {
	panic("TODO")
}

func (d *Destination) Abort() error {
	panic("TODO")
}
