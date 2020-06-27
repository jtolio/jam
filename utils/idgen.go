package utils

import (
	"crypto/rand"
	"encoding/base32"
)

var (
	// base32 standard encoding but lowercase
	encoding = base32.NewEncoding("abcdefghijklmnopqrstuvwxyz234567")
)

func IdBytesGen() []byte {
	// each digit in base32 is 5 bits, so we should use a multiple of
	// 5 bits to avoid wasted per-character entropy
	var buf [35]byte
	_, err := rand.Read(buf[:])
	if err != nil {
		panic(err)
	}
	return buf[:]
}

func IdGen() string {
	return PathSafeIdEncode(IdBytesGen())
}

func PathSafeIdEncode(data []byte) string {
	return encoding.EncodeToString(data)
}
