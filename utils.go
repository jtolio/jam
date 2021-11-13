package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/jtolio/jam/backends"
)

var (
	cmdBackendCat = &ffcli.Command{
		Name:       "backend-cat",
		ShortHelp:  "cat an object in the backend",
		ShortUsage: fmt.Sprintf("%s [opts] utils backend-cat <path-inside-backend>", os.Args[0]),
		Exec:       BackendCat,
	}

	cmdBackendSync = &ffcli.Command{
		Name:       "backend-sync",
		ShortHelp:  "sync one backend to another",
		ShortUsage: fmt.Sprintf("%s [opts] utils backend-sync <source-backend-url> <dest-backend-url>", os.Args[0]),
		Exec:       BackendSync,
	}

	cmdHashCoalesce = &ffcli.Command{
		Name:       "hash-coalesce",
		ShortHelp:  "combine hash files",
		ShortUsage: fmt.Sprintf("%s [opts] utils hash-coalesce", os.Args[0]),
		Exec:       HashCoalesce,
	}

	cmdHashSplit = &ffcli.Command{
		Name:       "hash-split",
		ShortHelp:  "split hash files to one per blob (default behavior on upload)",
		ShortUsage: fmt.Sprintf("%s [opts] utils hash-split", os.Args[0]),
		Exec:       HashSplit,
	}

	cmdUtils = &ffcli.Command{
		Name:       "utils",
		ShortHelp:  "miscellaneous utilities",
		ShortUsage: fmt.Sprintf("%s [opts] utils <subcommand> [opts]", os.Args[0]),
		Subcommands: []*ffcli.Command{
			cmdBackendCat,
			cmdBackendSync,
			cmdHashCoalesce,
			cmdHashSplit,
		},
		Exec: help,
	}
)

func BackendSync(ctx context.Context, args []string) error {
	if len(args) != 2 {
		return flag.ErrHelp
	}

	sourceSpec, err := url.Parse(args[0])
	if err != nil {
		return err
	}
	destSpec, err := url.Parse(args[1])
	if err != nil {
		return err
	}

	sourceStore, err := backends.Create(ctx, sourceSpec)
	if err != nil {
		return err
	}
	defer sourceStore.Close()

	destStore, err := backends.Create(ctx, destSpec)
	if err != nil {
		return err
	}
	defer destStore.Close()

	destContains := map[string]bool{}
	err = destStore.List(ctx, "", func(ctx context.Context, path string) error {
		destContains[path] = true
		return nil
	})
	if err != nil {
		return err
	}

	var missingPaths []string
	err = sourceStore.List(ctx, "", func(ctx context.Context, path string) error {
		if !destContains[path] {
			missingPaths = append(missingPaths, path)
		}
		return nil
	})
	if err != nil {
		return err
	}

	for _, path := range missingPaths {
		fmt.Printf("syncing %q\n", path)
		r, err := sourceStore.Get(ctx, path, 0, -1)
		if err != nil {
			return err
		}

		err = destStore.Put(ctx, path, r)
		r.Close()
		if err != nil {
			return err
		}
	}

	return nil
}

func BackendCat(ctx context.Context, args []string) error {
	if len(args) != 1 {
		return flag.ErrHelp
	}

	_, backend, _, mgrClose, err := getManager(ctx)
	if err != nil {
		return err
	}
	defer mgrClose()

	rc, err := backend.Get(ctx, args[0], 0, -1)
	if err != nil {
		return err
	}
	defer rc.Close()
	_, err = io.Copy(os.Stdout, rc)
	return err
}

func HashCoalesce(ctx context.Context, args []string) error {
	if len(args) != 0 {
		return flag.ErrHelp
	}

	_, _, hashes, mgrClose, err := getManager(ctx)
	if err != nil {
		return err
	}
	defer mgrClose()

	return hashes.Coalesce(ctx)
}

func HashSplit(ctx context.Context, args []string) error {
	if len(args) != 0 {
		return flag.ErrHelp
	}

	_, _, hashes, mgrClose, err := getManager(ctx)
	if err != nil {
		return err
	}
	defer mgrClose()

	return hashes.Split(ctx)
}

func byteFmt(bytes int64) string {
	val := float64(bytes)
	suffixes := []string{"B", "KB", "MB", "GB", "TB", "PB", "EB", "ZB", "YB"}
	for val > 1024 {
		val /= 1024
		suffixes = suffixes[1:]
	}
	return fmt.Sprintf("%0.02f %s", val, suffixes[0])
}
