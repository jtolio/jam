package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/jtolds/jam/backends/fs"
	"github.com/jtolds/jam/pkg/blobs"
	"github.com/jtolds/jam/pkg/enc"
	"github.com/jtolds/jam/pkg/mount"
	"github.com/jtolds/jam/pkg/session"
	"github.com/jtolds/jam/pkg/utils"
)

var (
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
	cmdStore = &ffcli.Command{
		Name:       "store",
		ShortHelp:  "store adds the given source directory to a new snapshot, forked from the latest",
		ShortUsage: fmt.Sprintf("%s store <source-dir> [<target-prefix>]", os.Args[0]),
		Exec:       Store,
	}
	cmdList = &ffcli.Command{
		Name:       "ls",
		ShortHelp:  "ls lists files in the given snapshot",
		ShortUsage: fmt.Sprintf("%s ls", os.Args[0]),
		Exec:       List,
	}
	cmdRoot = &ffcli.Command{
		ShortHelp:   "jam preserves your data",
		ShortUsage:  fmt.Sprintf("%s <subcommand>", os.Args[0]),
		Subcommands: []*ffcli.Command{cmdStore, cmdSnaps, cmdMount, cmdList},
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
	// TODO: make this all configurable!
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
		err = snapshot.List(ctx, "", "", func(ctx context.Context, entry *session.ListEntry) error {
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

func Store(ctx context.Context, args []string) error {
	if len(args) <= 0 || len(args) > 2 {
		return flag.ErrHelp
	}
	source := args[0]
	var targetPrefix string
	if len(args) == 2 {
		targetPrefix = args[1]
	}

	mgr, close, err := getManager(ctx)
	if err != nil {
		return err
	}
	defer close()

	sess, err := mgr.NewSession(ctx)
	if err != nil {
		return err
	}
	defer sess.Close()

	err = filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		base, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}

		fh, err := os.Open(path)
		if err != nil {
			return err
		}

		// PutFile closes the fh
		return sess.PutFile(ctx, targetPrefix+base, info.ModTime(), info.ModTime(), uint32(info.Mode()), fh)
	})
	if err != nil {
		return err
	}

	err = sess.Commit(ctx)
	if err != nil {
		return err
	}

	return nil
}

func Mount(ctx context.Context, args []string) error {
	// TODO: allow specification of other snapshots
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

	sess, err := mount.Mount(ctx, snap, args[0])
	if err != nil {
		return err
	}
	defer sess.Close()

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		sess.Close()
	}()

	sess.Wait()

	return nil
}

func List(ctx context.Context, args []string) error {
	// TODO: allow specification of other snapshots
	if len(args) != 0 {
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

	return snap.List(ctx, "", "", func(ctx context.Context, entry *session.ListEntry) error {
		fmt.Println(entry.Path)
		return nil
	})
}
