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
	"regexp"
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
	sysFlags      = flag.NewFlagSet("", flag.ExitOnError)
	sysFlagConfig = sysFlags.String("config", filepath.Join(homeDir(), ".jam", "jam.conf"),
		"path to config file")
	sysFlagBlockSize = sysFlags.Int("enc.block-size", 16*1024,
		"encryption block size")
	sysFlagPassphrase = sysFlags.String("enc.passphrase", "",
		"encryption passphrase")
	sysFlagStore = sysFlags.String("store", filepath.Join(homeDir(), ".jam", "storage"),
		"place to store data")
	sysFlagBlobSize = sysFlags.Int64("blobs.size", 64*1024*1024,
		"target blob size")
	sysFlagMaxUnflushed = sysFlags.Int("blobs.max-unflushed", 1000,
		"max number of objects to stage before flushing (requires file descriptor limit)")

	listFlags         = flag.NewFlagSet("", flag.ExitOnError)
	listFlagSnapshot  = listFlags.String("snap", "latest", "which snapshot to use")
	listFlagRecursive = listFlags.Bool("r", false, "list recursively")

	mountFlags        = flag.NewFlagSet("", flag.ExitOnError)
	mountFlagSnapshot = mountFlags.String("snap", "latest", "which snapshot to use")

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
		FlagSet:    mountFlags,
		Exec:       Mount,
	}
	cmdList = &ffcli.Command{
		Name:       "ls",
		ShortHelp:  "ls lists files in the given snapshot",
		ShortUsage: fmt.Sprintf("%s [opts] ls [opts] [<prefix>]", os.Args[0]),
		FlagSet:    listFlags,
		Exec:       List,
	}
	cmdStore = &ffcli.Command{
		Name:       "store",
		ShortHelp:  "store adds the given source directory to a new snapshot, forked from the latest",
		ShortUsage: fmt.Sprintf("%s [opts] store <source-dir> [<target-prefix>]", os.Args[0]),
		Exec:       Store,
	}
	cmdRename = &ffcli.Command{
		Name: "rename",
		ShortHelp: ("rename allows a regexp-based search and replace against all paths in the system, " +
			"forked from the latest snapshot. See https://golang.org/pkg/regexp/#Regexp.ReplaceAllString " +
			"for semantics."),
		ShortUsage: fmt.Sprintf("%s [opts] rename <regexp> <replacement>", os.Args[0]),
		Exec:       Rename,
	}
	cmdRoot = &ffcli.Command{
		ShortHelp:   "jam preserves your data",
		ShortUsage:  fmt.Sprintf("%s [opts] <subcommand> [opts]", os.Args[0]),
		Subcommands: []*ffcli.Command{cmdStore, cmdSnaps, cmdMount, cmdList, cmdRename},
		FlagSet:     sysFlags,
		Options: []ff.Option{
			ff.WithAllowMissingConfigFile(true),
			ff.WithConfigFileParser(ff.PlainParser),
			ff.WithConfigFileFlag("config"),
		},
		Exec: help,
	}
)

func homeDir() string {
	u, err := user.Current()
	if err != nil {
		panic(err)
	}
	if u.HomeDir == "" {
		panic("no homedir found")
	}
	return u.HomeDir
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
	if *sysFlagPassphrase == "" {
		return nil, nil, fmt.Errorf("invalid configuration, no root key specified")
	}
	backend := enc.NewEncWrapper(
		enc.NewSecretboxCodec(*sysFlagBlockSize),
		enc.NewHMACKeyGenerator([]byte(*sysFlagPassphrase)),
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

	return sess.Commit(ctx)
}

func Rename(ctx context.Context, args []string) error {
	if len(args) != 2 {
		return flag.ErrHelp
	}
	re, err := regexp.Compile(args[0])
	if err != nil {
		return err
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

	err = sess.Rename(ctx, re, args[1])
	if err != nil {
		return err
	}

	return sess.Commit(ctx)
}

func getReadSnapshot(ctx context.Context, mgr *session.Manager, snapshotFlag string) (session.Snapshot, error) {
	if snapshotFlag == "" || snapshotFlag == "latest" {
		return mgr.LatestSnapshot(ctx)
	}
	nano, err := strconv.ParseInt(snapshotFlag, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid snapshot value: %q", snapshotFlag)
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

	snap, err := getReadSnapshot(ctx, mgr, *mountFlagSnapshot)
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
	if len(args) != 0 && len(args) != 1 {
		return flag.ErrHelp
	}

	mgr, mgrClose, err := getManager(ctx)
	if err != nil {
		return err
	}
	defer mgrClose()

	snap, err := getReadSnapshot(ctx, mgr, *listFlagSnapshot)
	if err != nil {
		return err
	}
	defer snap.Close()

	delimiter := "/"
	if *listFlagRecursive {
		delimiter = ""
	}
	prefix := ""
	if len(args) > 0 {
		prefix = args[0]
	}

	return snap.List(ctx, prefix, delimiter, func(ctx context.Context, entry *session.ListEntry) error {
		fmt.Println(entry.Path)
		return nil
	})
}
