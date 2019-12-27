package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"time"

	"github.com/jtolds/jam/backends/fs"
	"github.com/jtolds/jam/pkg/blobs"
	"github.com/jtolds/jam/pkg/enc"
	"github.com/jtolds/jam/pkg/manifest"
	"github.com/jtolds/jam/pkg/session"
	"github.com/jtolds/jam/pkg/streams"
	"github.com/jtolds/jam/pkg/utils"
)

func main() {
	err := Main(context.Background())
	if err != nil {
		panic(fmt.Sprintf("%+v", err))
	}
}

func Main(ctx context.Context) error {
	backend := enc.NewEncWrapper(enc.NewSecretboxCodec(16*1024), enc.NewHMACKeyGenerator([]byte("hello")), fs.NewFS("test-data"))

	blobStore := blobs.NewStore(backend, 64*1024*1024, 1024)
	defer blobStore.Close()

	mgr := session.NewSessionManager(backend, utils.DefaultLogger, blobStore)
	session, err := mgr.NewSession(ctx)
	if err != nil {
		return err
	}
	defer session.Close()

	err = session.PutFile(ctx, "/etc/motd-"+fmt.Sprint(time.Now().Unix()), time.Now(), time.Now(), 0600, ioutil.NopCloser(bytes.NewReader([]byte("hello world\n"))))
	if err != nil {
		return err
	}
	err = session.Commit(ctx)
	if err != nil {
		return err
	}
	err = session.Close()
	if err != nil {
		return err
	}

	return mgr.ListSnapshots(ctx, func(ctx context.Context, timestamp time.Time) error {
		fmt.Println(timestamp)
		snapshot, err := mgr.OpenSnapshot(ctx, timestamp)
		if err != nil {
			return err
		}
		defer snapshot.Close()

		return snapshot.List(ctx, "", true, func(ctx context.Context, path string, metadata *manifest.Metadata, data *streams.Stream) error {
			defer data.Close()
			fmt.Println("  ", path, metadata)
			fmt.Println("===============")
			_, err := io.Copy(os.Stdout, data)
			fmt.Println("===============")
			return err
		})
	})
}
