package blobs

import (
	"io"

	"github.com/zeebo/errs"

	"github.com/jtolds/jam/pkg/manifest"
)

type concat struct {
	entries       []*entry
	current       *entry
	currentStream *manifest.Stream
	offset        int64
	blob          string
}

func newConcat(entries ...*entry) *concat {
	c := &concat{
		entries:       entries,
		currentStream: &manifest.Stream{},
		blob:          idGen(),
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
				return n, errs.Wrap(err)
			}
			c.advance()
			if c.current == nil {
				return n, io.EOF
			}
		}
		if n > 0 {
			return n, nil
		}
	}
}

func (c *concat) capRange() {
	if c.current != nil {
		rangeCount := len(c.currentStream.Ranges)
		length := c.offset -
			c.currentStream.Ranges[rangeCount-1].Offset
		if length > 0 {
			c.currentStream.Ranges[rangeCount-1].Length = length
		} else {
			c.currentStream.Ranges = c.currentStream.Ranges[:rangeCount-1]
		}
		c.current.cb(c.currentStream)
		c.currentStream = &manifest.Stream{}
	}
}

func (c *concat) advance() {
	c.capRange()
	if len(c.entries) == 0 {
		c.current = nil
	} else {
		c.current = c.entries[0]
		c.entries = c.entries[1:]
		c.currentStream = &manifest.Stream{
			Ranges: []*manifest.Range{
				&manifest.Range{
					Blob:   c.blob,
					Offset: c.offset,
				},
			},
		}
	}
}

func (c *concat) EOF() bool    { return c.current == nil }
func (c *concat) Blob() string { return c.blob }

func (c *concat) Cut() {
	c.capRange()
	c.offset = 0
	c.blob = idGen()
	if c.current != nil {
		c.currentStream.Ranges = append(c.currentStream.Ranges,
			&manifest.Range{
				Blob:   c.blob,
				Offset: c.offset,
			})
	}
}
