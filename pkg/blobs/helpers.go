package blobs

import (
	"crypto/rand"
	"encoding/base64"
)

func idGen() string {
	var buf [32]byte
	_, err := rand.Read(buf[:])
	if err != nil {
		panic(err)
	}
	return base64.URLEncoding.EncodeToString(buf[:])
}
