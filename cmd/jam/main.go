package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"os/user"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	ff "github.com/peterbourgon/ff/v3"
	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/jtolds/jam/backends/fs"
	"github.com/jtolds/jam/pkg/blobs"
	"github.com/jtolds/jam/pkg/enc"
	"github.com/jtolds/jam/pkg/mount"
	"github.com/jtolds/jam/pkg/session"
	"github.com/jtolds/jam/pkg/utils"
)

var (
	sysFlags            = flag.NewFlagSet("", flag.ExitOnError)
	sysFlagConfig       = sysFlags.String("config", defaultConfigFile(), "path to config file")
	sysFlagBlockSize    = sysFlags.Int("enc.block-size", 16*1024, "encryption block size")
	sysFlagRootKey      = sysFlags.String("enc.root-key", "", "root encryption key")
	sysFlagStore        = sysFlags.String("store", "test-data", "place to store data")
	sysFlagBlobSize     = sysFlags.Int64("blobs.size", 64*1024*1024, "target blob size")
	sysFlagMaxUnflushed = sysFlags.Int("blobs.max-unflushed", 1000,
		"max number of objects to stage before flushing (requires file descriptor limit)")

	readFlags        = flag.NewFlagSet("", flag.ExitOnError)
	readFlagSnapshot = readFlags.String("snap", "latest", "which snapshot to use")

	cmdSnaps = &ffcli.Command{
		Name:       "snaps",
		ShortHelp:  "lists snapshots",
		ShortUsage: fmt.Sprintf("%s [opts] snaps", os.Args[0]),
		Exec:       Snaps,
	}
	cmdMount = &ffcli.Command{
		Name:       "mount",
		ShortHelp:  "mounts snap as read-only filesystem",
		ShortUsage: fmt.Sprintf("%s [opts] mount [opts] <target>", os.Args[0]),
		FlagSet:    readFlags,
		Exec:       Mount,
	}
	cmdList = &ffcli.Command{
		Name:       "ls",
		ShortHelp:  "ls lists files in the given snapshot",
		ShortUsage: fmt.Sprintf("%s [opts] ls [opts]", os.Args[0]),
		FlagSet:    readFlags,
		Exec:       List,
	}
	cmdStore = &ffcli.Command{
		Name:       "store",
		ShortHelp:  "store adds the given source directory to a new snapshot, forked from the latest",
		ShortUsage: fmt.Sprintf("%s [opts] store <source-dir> [<target-prefix>]", os.Args[0]),
		Exec:       Store,
	}
	cmdRoot = &ffcli.Command{
		ShortHelp:   "jam preserves your data",
		ShortUsage:  fmt.Sprintf("%s [opts] <subcommand> [opts]", os.Args[0]),
		Subcommands: []*ffcli.Command{cmdStore, cmdSnaps, cmdMount, cmdList},
		FlagSet:     sysFlags,
		Options: []ff.Option{
			ff.WithAllowMissingConfigFile(true),
			ff.WithConfigFileParser(ff.PlainParser),
			ff.WithConfigFileFlag("config"),
		},
		Exec: help,
	}
)

func defaultConfigFile() string {
	u, err := user.Current()
	if err != nil {
		panic(err)
	}
	if u.HomeDir == "" {
		panic("no homedir found")
	}
	return filepath.Join(u.HomeDir, ".jam", "jam.conf")
}

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
	if *sysFlagRootKey == "" {
		return nil, nil, fmt.Errorf("invalid configuration, no root key specified")
	}
	backend := enc.NewEncWrapper(
		enc.NewSecretboxCodec(*sysFlagBlockSize),
		enc.NewHMACKeyGenerator([]byte(*sysFlagRootKey)),
		fs.NewFS(*sysFlagStore),
	)
	blobs := blobs.NewStore(backend, *sysFlagBlobSize, *sysFlagMaxUnflushed)
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

func getReadSnapshot(ctx context.Context, mgr *session.Manager) (session.Snapshot, error) {
	if *readFlagSnapshot == "" || *readFlagSnapshot == "latest" {
		return mgr.LatestSnapshot(ctx)
	}
	nano, err := strconv.ParseInt(*readFlagSnapshot, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid snapshot value: %q", *readFlagSnapshot)
	}
	return mgr.OpenSnapshot(ctx, time.Unix(0, nano))
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

	snap, err := getReadSnapshot(ctx, mgr)
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

	return sess.Serve()
}

func List(ctx context.Context, args []string) error {
	if len(args) != 0 {
		return flag.ErrHelp
	}

	mgr, mgrClose, err := getManager(ctx)
	if err != nil {
		return err
	}
	defer mgrClose()

	snap, err := getReadSnapshot(ctx, mgr)
	if err != nil {
		return err
	}
	defer snap.Close()

	return snap.List(ctx, "", "", func(ctx context.Context, entry *session.ListEntry) error {
		fmt.Println(entry.Path)
		return nil
	})
}
