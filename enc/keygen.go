package enc

import (
	"crypto/hmac"
	"crypto/sha256"
)

// A KeyGenerator determines the encryption key to use for a given
// backend path
type KeyGenerator interface {
	KeyForPath(path string) [32]byte
}

// HMACKeyGenerator is a KeyGenerator that simply SHA256-HMACs the
// provided root key with any given full path.
type HMACKeyGenerator struct {
	key []byte
}

var _ KeyGenerator = (*HMACKeyGenerator)(nil)

func NewHMACKeyGenerator(rootKey []byte) *HMACKeyGenerator {
	return &HMACKeyGenerator{key: rootKey}
}

func (s *HMACKeyGenerator) KeyForPath(path string) (key [32]byte) {
	mac := hmac.New(sha256.New, s.key)
	_, err := mac.Write([]byte(path))
	if err != nil {
		panic(err)
	}
	if len(mac.Sum(key[:0])) != len(key[:]) {
		panic("hmac output failure")
	}
	return key
}
