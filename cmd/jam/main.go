package main

import (
	"context"
	"fmt"
	"time"

	"github.com/jtolds/jam/backends/fs"
	"github.com/jtolds/jam/pkg/blobs"
	"github.com/jtolds/jam/pkg/enc"
	"github.com/jtolds/jam/pkg/session"
	"github.com/jtolds/jam/pkg/utils"
)

func main() {
	ctx := context.Background()
	backend := enc.NewEncWrapper(enc.NewSecretboxCodec(16*1024), enc.NewHMACKeyGenerator([]byte("hello")), fs.NewFS("."))
	blobStore := blobs.NewStore(backend, 64*1024*1024, 1024)
	mgr := session.NewSessionManager(backend, utils.DefaultLogger, blobStore)
	err := mgr.ListSnapshots(ctx, func(ctx context.Context, timestamp time.Time) error {
		_, err := fmt.Println(timestamp)
		return err
	})
	if err != nil {
		panic(err)
	}
}
