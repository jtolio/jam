package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/jtolds/jam/manifest"
	"github.com/jtolds/jam/session"
	"github.com/jtolds/jam/streams"
	"github.com/jtolds/jam/utils"
	"github.com/peterbourgon/ff/v3/ffcli"
)

var (
	integrityFlags            = flag.NewFlagSet("", flag.ExitOnError)
	integrityFlagSnapshot     = integrityFlags.String("snap", "latest", "which snapshot to use")
	integrityFlagShowUnneeded = integrityFlags.Bool("show-unneeded", false, "if true, show unneeded blobs")
	integrityFlagSkipBlobEnd  = integrityFlags.Bool("skip-blob-end", false, "if true, skip trying to read the known end of each blob")

	cmdIntegrity = &ffcli.Command{
		Name:       "integrity",
		ShortHelp:  "integrity check. for full effect, disable caching and enable read\n\tcomparison",
		ShortUsage: fmt.Sprintf("%s [opts] integrity [opts]", os.Args[0]),
		FlagSet:    integrityFlags,
		Exec:       Integrity,
	}
)

func Integrity(ctx context.Context, args []string) error {
	if len(args) != 0 {
		return flag.ErrHelp
	}

	mgr, backend, hashes, mgrClose, err := getManager(ctx)
	if err != nil {
		return err
	}
	defer mgrClose()

	blobs := map[string]bool{}
	blobLastRange := map[string]*manifest.Range{}
	missing := map[string]bool{}
	bad := map[string]bool{}

	utils.L(ctx).Debugf("confirming that a blob exists for every hash")

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
			blobPath := streams.BlobPath(r.Blob())
			if lastRange, exists := blobLastRange[blobPath]; r.Length > 0 && (!exists || lastRange.Offset < r.Offset) {
				blobLastRange[blobPath] = r
			}
			if !blobs[blobPath] {
				if !missing[r.Blob()] {
					missing[r.Blob()] = true
					fmt.Printf("missing blob: %s\n", r.Blob())
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

	utils.L(ctx).Debugf("no dangling hashes")

	if *integrityFlagShowUnneeded {
		for path := range blobs {
			if _, exists := blobLastRange[path]; !exists {
				fmt.Printf("blob unnecessary: %s\n", path)
			}
		}
	}

	utils.L(ctx).Debugf("make sure a hash for every listed path exists")

	// check to make sure a hash for every listed path exists
	snap, _, err := getReadSnapshot(ctx, mgr, *integrityFlagSnapshot)
	if err != nil {
		return err
	}
	defer snap.Close()
	err = snap.List(ctx, "", "", func(ctx context.Context, entry *session.ListEntry) error {
		if entry.Meta.Type != manifest.Metadata_FILE {
			return nil
		}
		stream, err := entry.Stream(ctx)
		if err != nil {
			return err
		}
		return stream.Close()
	})
	if err != nil {
		return err
	}

	utils.L(ctx).Debugf("no dangling paths")

	// check to make sure none of the blobs are truncated
	if !*integrityFlagSkipBlobEnd {
		utils.L(ctx).Debugf("checking to make sure the last byte of each blob is readable")

		for path, r := range blobLastRange {
			utils.L(ctx).Debugf("checking end of %q, %d", path, r.Offset+r.Length)
			rc, err := streams.OpenRange(ctx, backend, r, r.Length-1)
			if err != nil {
				return err
			}
			// authenticated encryption will throw an error if the data is bad
			_, err = io.Copy(ioutil.Discard, rc)
			if err != nil {
				rc.Close()
				return err
			}
			err = rc.Close()
			if err != nil {
				return err
			}
		}

		utils.L(ctx).Debugf("looks good")
	}

	return nil
}
