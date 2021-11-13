package blobs

import (
	"context"
	"io"

	"github.com/zeebo/errs"

	"github.com/jtolio/jam/manifest"
	"github.com/jtolio/jam/utils"
)

type concat struct {
	unprocessed []*entry
	processing  *entry
	processed   []*entry

	stagedRange *manifest.Range

	offset int64
	blob   []byte
}

func newConcat(entries ...*entry) *concat {
	c := &concat{
		unprocessed: entries,
	}
	c.cut()
	c.advance()
	return c
}

func (c *concat) advance() {
	if c.processing != nil {
		c.capRange()
		c.processed = append(c.processed, c.processing)
		c.processing = nil
	}
	if len(c.unprocessed) > 0 {
		c.processing = c.unprocessed[0]
		c.processing.stream = &manifest.Stream{}
		c.unprocessed = c.unprocessed[1:]
		c.resetRange()
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
		c.processing.stream.Ranges = append(c.processing.stream.Ranges, c.stagedRange)
	}
	c.stagedRange = nil
}

func (c *concat) resetRange() {
	c.stagedRange = &manifest.Range{
		BlobBytes: c.blob,
		Offset:    c.offset,
	}
}

func (c *concat) EOF() bool    { return c.processing == nil }
func (c *concat) Blob() string { return utils.PathSafeIdEncode(c.blob) }

func (c *concat) cut() {
	if c.processing != nil {
		c.capRange()
		defer c.resetRange()
	}
	c.offset = 0
	c.blob = utils.IdBytesGen()
}

func (c *concat) Cut(ctx context.Context) error {
	c.cut()
	for i, entry := range c.processed {
		err := entry.cb(ctx, entry.stream, i == len(c.processed)-1)
		if err != nil {
			return err
		}
	}
	c.processed = nil
	return nil
}
