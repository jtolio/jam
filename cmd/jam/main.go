package main

import (
	"github.com/jtolds/jam/backends/fs"
	"github.com/jtolds/jam/enc"
)

func main() {
	enc.NewEncWrapper(nil, fs.NewFS("."))
}
