package main

import (
	"github.com/jtolds/jam/backends/fs"
	"github.com/jtolds/jam/pkg/enc"
)

func main() {
	enc.NewEncWrapper(enc.NewSecretboxCodec(16*1024), enc.NewHMACKeyGenerator([]byte("hello")), fs.NewFS("."))
}
