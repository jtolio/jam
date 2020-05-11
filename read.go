package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sync/atomic"
	"syscall"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/jtolds/jam/mount"
	"github.com/jtolds/jam/session"
)

var (
	listFlags         = flag.NewFlagSet("", flag.ExitOnError)
	listFlagSnapshot  = listFlags.String("snap", "latest", "which snapshot to use")
	listFlagRecursive = listFlags.Bool("r", false, "list recursively")

	mountFlags        = flag.NewFlagSet("", flag.ExitOnError)
	mountFlagSnapshot = mountFlags.String("snap", "latest", "which snapshot to use")

	cmdMount = &ffcli.Command{
		Name:       "mount",
		ShortHelp:  "mounts snap as read-only filesystem",
		ShortUsage: fmt.Sprintf("%s [opts] mount [opts] <target>", os.Args[0]),
		FlagSet:    mountFlags,
		Exec:       Mount,
	}
	cmdLs = &ffcli.Command{
		Name:       "ls",
		ShortHelp:  "ls lists files in the given snapshot",
		ShortUsage: fmt.Sprintf("%s [opts] ls [opts] [<prefix>]", os.Args[0]),
		FlagSet:    listFlags,
		Exec:       List,
	}
)

func Mount(ctx context.Context, args []string) error {
	if len(args) != 1 {
		return flag.ErrHelp
	}
	mountpoint, err := filepath.Abs(args[0])
	if err != nil {
		return err
	}

	mgr, mgrClose, err := getManager(ctx)
	if err != nil {
		return err
	}
	defer mgrClose()

	snap, ts, err := getReadSnapshot(ctx, mgr, *mountFlagSnapshot)
	if err != nil {
		return err
	}
	defer snap.Close()

	sess, err := mount.Mount(ctx, snap, mountpoint)
	if err != nil {
		return err
	}
	defer sess.Close()

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	canceled := new(int32)
	go func() {
		for range c {
			if atomic.CompareAndSwapInt32(canceled, 0, 1) {
				fmt.Printf("\runmounting %q\n", mountpoint)
			}
			sess.Close()
		}
	}()

	fmt.Printf("mounted snapshot %d at %q\n", ts.UnixNano(), mountpoint)
	err = sess.Serve()
	if atomic.LoadInt32(canceled) == 1 {
		err = nil
	}
	return err
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

	snap, _, err := getReadSnapshot(ctx, mgr, *listFlagSnapshot)
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
		if entry.Prefix {
			fmt.Println(entry.Path + "/")
		} else {
			fmt.Println(entry.Path)
		}
		return nil
	})
}
