package blobs

import (
	"io"

	"github.com/zeebo/errs"

	"github.com/jtolds/jam/manifest"
)

type concat struct {
	unprocessed []*entry
	processing  *entry

	stagedStream *manifest.Stream
	stagedRange  *manifest.Range

	offset int64
	blob   string
}

func newConcat(entries ...*entry) *concat {
	c := &concat{
		unprocessed: entries,
	}
	c.Cut()
	c.advance()
	return c
}

func (c *concat) advance() {
	if c.processing != nil {
		c.capRange()
		c.processing.cb(c.stagedStream)
		c.stagedStream = nil
	}
	if len(c.unprocessed) > 0 {
		c.processing = c.unprocessed[0]
		c.unprocessed = c.unprocessed[1:]
		c.stagedStream = &manifest.Stream{}
		c.resetRange()
	} else {
		c.processing = nil
	}
}

func (c *concat) Read(p []byte) (n int, err error) {
	if c.processing == nil {
		return 0, io.EOF
	}
	for {
		n, err = c.processing.source.Read(p)
		c.offset += int64(n)
		if err != nil {
			if err != io.EOF {
				return n, errs.Wrap(err)
			}
			c.advance()
			if c.processing == nil {
				return n, io.EOF
			}
		}
		if n > 0 {
			return n, nil
		}
	}
}

func (c *concat) capRange() {
	c.stagedRange.Length = c.offset - c.stagedRange.Offset
	if c.stagedRange.Length > 0 {
		c.stagedStream.Ranges = append(c.stagedStream.Ranges, c.stagedRange)
	}
	c.stagedRange = nil
}

func (c *concat) resetRange() {
	c.stagedRange = &manifest.Range{
		Blob:   c.blob,
		Offset: c.offset,
	}
}

func (c *concat) EOF() bool    { return c.processing == nil }
func (c *concat) Blob() string { return c.blob }

func (c *concat) Cut() {
	if c.processing != nil {
		c.capRange()
		defer c.resetRange()
	}
	c.offset = 0
	c.blob = idGen()
}
