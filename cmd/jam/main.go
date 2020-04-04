package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"time"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/jtolds/jam/backends/fs"
	"github.com/jtolds/jam/pkg/blobs"
	"github.com/jtolds/jam/pkg/enc"
	"github.com/jtolds/jam/pkg/manifest"
	"github.com/jtolds/jam/pkg/mount"
	"github.com/jtolds/jam/pkg/session"
	"github.com/jtolds/jam/pkg/streams"
	"github.com/jtolds/jam/pkg/utils"
)

var (
	cmdTest = &ffcli.Command{
		Name: "test",
		Exec: Test,
	}
	cmdSnaps = &ffcli.Command{
		Name:       "snaps",
		ShortHelp:  "lists snapshots",
		ShortUsage: fmt.Sprintf("%s snaps", os.Args[0]),
		Exec:       Snaps,
	}
	cmdMount = &ffcli.Command{
		Name:       "mount",
		ShortHelp:  "mounts snap as read-only filesystem",
		ShortUsage: fmt.Sprintf("%s mount <target>", os.Args[0]),
		Exec:       Mount,
	}
	cmdRoot = &ffcli.Command{
		ShortHelp:   "jam preserves your data",
		ShortUsage:  fmt.Sprintf("%s <subcommand>", os.Args[0]),
		Subcommands: []*ffcli.Command{cmdTest, cmdSnaps, cmdMount},
		Exec:        help,
	}
)

func help(ctx context.Context, args []string) error { return flag.ErrHelp }

func main() {
	err := cmdRoot.ParseAndRun(context.Background(), os.Args[1:])
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return
		}
		fmt.Fprintf(os.Stderr, "error: %+v\n", err)
		os.Exit(1)
	}
}

func getManager(ctx context.Context) (mgr *session.Manager, close func() error, err error) {
	backend := enc.NewEncWrapper(
		enc.NewSecretboxCodec(16*1024),
		enc.NewHMACKeyGenerator([]byte("hello")),
		fs.NewFS("test-data"),
	)
	blobs := blobs.NewStore(backend, 64*1024*1024, 1024)
	return session.NewManager(utils.DefaultLogger, backend, blobs), blobs.Close, nil
}

func Snaps(ctx context.Context, args []string) error {
	if len(args) != 0 {
		return flag.ErrHelp
	}

	mgr, mgrClose, err := getManager(ctx)
	if err != nil {
		return err
	}
	defer mgrClose()

	return mgr.ListSnapshots(ctx, func(ctx context.Context, timestamp time.Time) error {
		snapshot, err := mgr.OpenSnapshot(ctx, timestamp)
		if err != nil {
			return err
		}
		defer snapshot.Close()
		var fileCount int64
		err = snapshot.List(ctx, "", "", func(ctx context.Context, path string, metadata *manifest.Metadata, data *streams.Stream) error {
			defer data.Close()
			fileCount++
			return nil
		})
		if err != nil {
			return err
		}

		fmt.Printf("%v: %v (%d files)\n", timestamp.UnixNano(), timestamp.Local().Format("2006-01-02 03:04:05 pm"), fileCount)
		return nil
	})
}

func Test(ctx context.Context, args []string) error {
	if len(args) != 0 {
		return flag.ErrHelp
	}

	mgr, close, err := getManager(ctx)
	if err != nil {
		return err
	}
	defer close()

	session, err := mgr.NewSession(ctx)
	if err != nil {
		return err
	}

	err = session.PutFile(ctx, "/etc/motd-"+fmt.Sprint(time.Now().Unix()), time.Now(), time.Now(), 0600, ioutil.NopCloser(bytes.NewReader([]byte("hello world\n"))))
	if err != nil {
		session.Close()
		return err
	}
	err = session.Commit(ctx)
	if err != nil {
		session.Close()
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

		return snapshot.List(ctx, "", "", func(ctx context.Context, path string, metadata *manifest.Metadata, data *streams.Stream) error {
			defer data.Close()
			fmt.Println("  ", path, metadata)
			fmt.Println("===============")
			_, err := io.Copy(os.Stdout, data)
			fmt.Println("===============")
			return err
		})
	})
}

func Mount(ctx context.Context, args []string) error {
	if len(args) != 1 {
		return flag.ErrHelp
	}

	mgr, mgrClose, err := getManager(ctx)
	if err != nil {
		return err
	}
	defer mgrClose()

	snap, err := mgr.LatestSnapshot(ctx)
	if err != nil {
		return err
	}
	defer snap.Close()

	return mount.Mount(ctx, snap, args[0])
}
