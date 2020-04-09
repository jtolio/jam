package enc

import (
	"fmt"

	"golang.org/x/crypto/nacl/secretbox"
)

// SecretboxCodec provides the NaCl 'secretbox' encryption scheme
// as a Codec. Nonces are monotonic with blocks starting with 0.
type SecretboxCodec struct {
	unencryptedBlockSize int
}

var _ Codec = (*SecretboxCodec)(nil)

// NewSecretboxCodec creates a SecretboxCodec with the given unencrypted
// block size. A good choice here is 16*1024, or 16*1024-secretbox.Overhead,
// depending on your alignment needs.
func NewSecretboxCodec(unencryptedBlockSize int) *SecretboxCodec {
	return &SecretboxCodec{unencryptedBlockSize: unencryptedBlockSize}
}

func (s *SecretboxCodec) DecodedBlockSize() int { return s.unencryptedBlockSize }
func (s *SecretboxCodec) EncodedBlockSize() int { return s.unencryptedBlockSize + secretbox.Overhead }

func (s *SecretboxCodec) Encode(out, in []byte, key *[32]byte, blockNum int64) ([]byte, error) {
	return secretbox.Seal(out, in, calcNonce(blockNum), key), nil
}

func (s *SecretboxCodec) Decode(out, in []byte, key *[32]byte, blockNum int64) ([]byte, error) {
	rv, success := secretbox.Open(out, in, calcNonce(blockNum), key)
	if !success {
		return nil, fmt.Errorf("failed decrypting")
	}
	return rv, nil
}
