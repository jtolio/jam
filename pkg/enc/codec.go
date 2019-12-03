package enc

// A Codec concisely represents a reversible transformation that might
// be applied to a data stream, such as encryption.
type Codec interface {
	EncodedBlockSize() int
	DecodedBlockSize() int
	Encode(out, in []byte, key *[32]byte, blockNum int64) ([]byte, error)
	Decode(out, in []byte, key *[32]byte, blockNum int64) ([]byte, error)
}

type reversedCodec struct {
	c Codec
}

// Reverse returns a new Codec with the opposite data transformation of
// the provided Codec
func Reverse(c Codec) Codec {
	return reversedCodec{c: c}
}

func (r reversedCodec) EncodedBlockSize() int { return r.c.DecodedBlockSize() }
func (r reversedCodec) DecodedBlockSize() int { return r.c.EncodedBlockSize() }

func (r reversedCodec) Encode(out, in []byte, key *[32]byte, blockNum int64) ([]byte, error) {
	return r.c.Decode(out, in, key, blockNum)
}

func (r reversedCodec) Decode(out, in []byte, key *[32]byte, blockNum int64) ([]byte, error) {
	return r.c.Encode(out, in, key, blockNum)
}
