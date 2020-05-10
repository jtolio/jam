package blobs

import (
	"context"
	"io"

	"github.com/zeebo/errs"

	"github.com/jtolds/jam/manifest"
)

type concat struct {
	ctx context.Context

	unprocessed []*entry
	processing  *entry

	stagedStream *manifest.Stream
	stagedRange  *manifest.Range

	offset int64
	blob   string
}

func newConcat(ctx context.Context, entries ...*entry) (*concat, error) {
	c := &concat{
		ctx:         ctx,
		unprocessed: entries,
	}
	c.Cut()
	return c, c.advance()
}

func (c *concat) advance() (err error) {
	if c.processing != nil {
		c.capRange()
		err = c.processing.cb(c.ctx, c.stagedStream)
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
	return err
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
			err = c.advance()
			if err != nil {
				return n, err
			}
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
	c.blob = IdGen()
}
