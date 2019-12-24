package session

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

func blobPath(id string) string {
	return "blob/" + id[:2] + "/" + id[2:]
}
