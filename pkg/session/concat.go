package session

import (
	"io"

	"github.com/jtolds/jam/pkg/manifest"
)

type concat struct {
	entries []*stagedEntry
	current *stagedEntry
	offset  int64
	blob    string
}

func newConcat(entries ...*stagedEntry) *concat {
	c := &concat{
		entries: entries,
		blob:    idGen(),
	}
	c.advance()
	return c
}

func (c *concat) Read(p []byte) (n int, err error) {
	if c.current == nil {
		return 0, io.EOF
	}
	for {
		n, err = c.current.source.Read(p)
		c.offset += int64(n)
		if err != nil {
			if err != io.EOF {
				return n, err
			}
			c.advance()
			if c.current == nil {
				return n, err
			}
		}
		if n > 0 {
			return n, nil
		}
	}
}

func (c *concat) capRange() {
	if c.current != nil {
		rangeCount := len(c.current.stream.Ranges)
		length := c.offset -
			c.current.stream.Ranges[rangeCount-1].Offset
		if length > 0 {
			c.current.stream.Ranges[rangeCount-1].Length = length
		} else {
			c.current.stream.Ranges = c.current.stream.Ranges[:rangeCount-1]
		}
	}
}

func (c *concat) advance() {
	c.capRange()
	if len(c.entries) == 0 {
		c.current = nil
	} else {
		c.current = c.entries[0]
		c.entries = c.entries[1:]
		c.current.stream = &manifest.Stream{
			Ranges: []*manifest.Range{
				&manifest.Range{
					Blob:   c.blob,
					Offset: c.offset,
				}}}
	}
}

func (c *concat) EOF() bool    { return c.current == nil }
func (c *concat) Blob() string { return c.blob }

func (c *concat) Cut() {
	c.capRange()
	c.offset = 0
	c.blob = idGen()
	if c.current != nil {
		c.current.stream.Ranges = append(c.current.stream.Ranges,
			&manifest.Range{
				Blob:   c.blob,
				Offset: c.offset,
			})
	}
}
