package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/jtolds/jam/session"
)

var (
	cmdSnaps = &ffcli.Command{
		Name:       "snaps",
		ShortHelp:  "lists snapshots",
		ShortUsage: fmt.Sprintf("%s [opts] snaps", os.Args[0]),
		Exec:       Snaps,
	}
	cmdUnsnap = &ffcli.Command{
		Name:       "unsnap",
		ShortHelp:  "unsnap removes an old snap",
		ShortUsage: fmt.Sprintf("%s [opts] unsnap <snapid>", os.Args[0]),
		Exec:       Unsnap,
	}
	cmdRevertTo = &ffcli.Command{
		Name:       "revert-to",
		ShortHelp:  "revert-to makes a new snapshot that matches an older one",
		ShortUsage: fmt.Sprintf("%s [opts] revert-to <snapid>", os.Args[0]),
		Exec:       RevertTo,
	}
)

func Snaps(ctx context.Context, args []string) error {
	if len(args) != 0 {
		return flag.ErrHelp
	}

	mgr, _, _, mgrClose, err := getManager(ctx)
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

func Unsnap(ctx context.Context, args []string) error {
	if len(args) != 1 {
		return flag.ErrHelp
	}

	mgr, _, _, mgrClose, err := getManager(ctx)
	if err != nil {
		return err
	}
	defer mgrClose()

	nano, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid snapshot value: %q", args[0])
	}
	return mgr.DeleteSnapshot(ctx, time.Unix(0, nano))
}

func RevertTo(ctx context.Context, args []string) error {
	if len(args) != 1 {
		return flag.ErrHelp
	}

	mgr, _, _, mgrClose, err := getManager(ctx)
	if err != nil {
		return err
	}
	defer mgrClose()

	nano, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid snapshot value: %q", args[0])
	}

	sess, err := mgr.RevertTo(ctx, time.Unix(0, nano))
	if err != nil {
		return err
	}
	defer sess.Close()
	return sess.Commit(ctx)
}
