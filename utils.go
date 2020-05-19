package main

import (
	"context"
	"flag"
	"fmt"
	"net/url"
	"os"

	"github.com/jtolds/jam/backends"
	"github.com/jtolds/jam/manifest"
	"github.com/jtolds/jam/streams"
	"github.com/peterbourgon/ff/v3/ffcli"
)

var (
	cmdBackendSync = &ffcli.Command{
		Name:       "backend-sync",
		ShortHelp:  "sync one backend to another",
		ShortUsage: fmt.Sprintf("%s [opts] utils backend-sync <source-backend-url> <dest-backend-url>", os.Args[0]),
		Exec:       BackendSync,
	}

	cmdHashCheck = &ffcli.Command{
		Name:       "hash-check",
		ShortHelp:  "makes sure blobs exist for all hashes",
		ShortUsage: fmt.Sprintf("%s [opts] utils hash-check", os.Args[0]),
		Exec:       HashCheck,
	}

	cmdHashCoalesce = &ffcli.Command{
		Name:       "hash-coalesce",
		ShortHelp:  "combine hash files",
		ShortUsage: fmt.Sprintf("%s [opts] utils hash-coalesce", os.Args[0]),
		Exec:       HashCoalesce,
	}

	cmdUtils = &ffcli.Command{
		Name:       "utils",
		ShortHelp:  "miscellaneous utilities",
		ShortUsage: fmt.Sprintf("%s [opts] utils <subcommand> [opts]", os.Args[0]),
		Subcommands: []*ffcli.Command{
			cmdBackendSync,
			cmdHashCheck,
			cmdHashCoalesce,
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

func HashCheck(ctx context.Context, args []string) error {
	if len(args) != 0 {
		return flag.ErrHelp
	}

	_, backend, hashes, mgrClose, err := getManager(ctx)
	if err != nil {
		return err
	}
	defer mgrClose()

	blobs := map[string]bool{}
	missing := map[string]bool{}
	bad := map[string]bool{}

	err = backend.List(ctx, streams.BlobPrefix,
		func(ctx context.Context, path string) error {
			blobs[path] = true
			return nil
		})
	if err != nil {
		return err
	}

	err = hashes.Iterate(ctx, func(ctx context.Context, hash, hashset string, stream *manifest.Stream) error {
		for _, r := range stream.Ranges {
			if !blobs[streams.BlobPath(r.Blob)] {
				if !missing[r.Blob] {
					missing[r.Blob] = true
					fmt.Printf("missing blob: %s\n", r.Blob)
				}
				if !bad[hashset] {
					bad[hashset] = true
					fmt.Printf("from hash set: %s\n", hashset)
				}
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	return nil
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
