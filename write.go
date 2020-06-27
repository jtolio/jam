package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/jtolds/jam/utils"
)

var (
	cmdStore = &ffcli.Command{
		Name:       "store",
		ShortHelp:  "store adds the given source directory to a new snapshot, forked\n\tfrom the latest snapshot.",
		ShortUsage: fmt.Sprintf("%s [opts] store <source-dir> [<target-prefix>]", os.Args[0]),
		Exec:       Store,
	}
	cmdRename = &ffcli.Command{
		Name: "rename",
		ShortHelp: ("rename allows a regexp-based search and replace against all paths\n\tin the system, " +
			"forked from the latest snapshot. See\n\thttps://golang.org/pkg/regexp/#Regexp.ReplaceAll " +
			"for semantics."),
		ShortUsage: fmt.Sprintf("%s [opts] rename <regexp> <replacement>", os.Args[0]),
		Exec:       Rename,
	}
	cmdRm = &ffcli.Command{
		Name: "rm",
		ShortHelp: ("rm deletes all paths that match the provided regexp.\n\t" +
			"https://golang.org/pkg/regexp/#Regexp.Match for semantics."),
		ShortUsage: fmt.Sprintf("%s [opts] rm <regexp>", os.Args[0]),
		Exec:       Remove,
	}
)

func Store(ctx context.Context, args []string) error {
	if len(args) <= 0 || len(args) > 2 {
		return flag.ErrHelp
	}
	source := args[0]
	var targetPrefix string
	if len(args) == 2 {
		targetPrefix = args[1]
	}

	mgr, _, _, close, err := getManager(ctx)
	if err != nil {
		return err
	}
	defer close()

	sess, err := mgr.NewSession(ctx)
	if err != nil {
		return err
	}
	defer sess.Close()

	// TODO: don't abort the entire walk when just one file fails
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

		if info.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(path)
			if err != nil {
				return err
			}
			return sess.PutSymlink(ctx, targetPrefix+base, info.ModTime(), info.ModTime(), uint32(info.Mode()), target)
		}

		if !info.Mode().IsRegular() {
			utils.L(ctx).Normalf("skipping %q, mode type not understood", targetPrefix+base)
			return nil
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

	mgr, _, _, close, err := getManager(ctx)
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

func Remove(ctx context.Context, args []string) error {
	if len(args) != 1 {
		return flag.ErrHelp
	}
	re, err := regexp.Compile(args[0])
	if err != nil {
		return err
	}

	mgr, _, _, close, err := getManager(ctx)
	if err != nil {
		return err
	}
	defer close()

	sess, err := mgr.NewSession(ctx)
	if err != nil {
		return err
	}
	defer sess.Close()

	err = sess.DeleteAll(ctx, re)
	if err != nil {
		return err
	}

	return sess.Commit(ctx)
}
