package hashdb

import (
	"context"

	"github.com/jtolds/jam/manifest"
	"github.com/jtolds/jam/utils"
)

func (d *DB) Coalesce(ctx context.Context) error {
	// TODO: seems silly to write out a small hashset only to delete it
	err := d.Flush(ctx)
	if err != nil {
		return err
	}
	newpath, err := d.flush(ctx, d.existing)
	if err != nil {
		return err
	}
	utils.L(ctx).Normalf("wrote new hashset with %d hashes. deleting old hashsets...", len(d.existing))
	deleted := map[string]bool{}
	for _, oldpath := range d.paths {
		if deleted[oldpath] {
			continue
		}
		err = d.backend.Delete(ctx, oldpath)
		if err != nil {
			return err
		}
		deleted[oldpath] = true
	}
	d.paths = []string{newpath}
	for hash := range d.source {
		d.source[hash] = newpath
	}
	utils.L(ctx).Normalf("deleted %d old hashsets.", len(deleted))
	return nil
}

func (d *DB) Split(ctx context.Context) error {
	// TODO: seems silly to write out a small hashset only to delete it
	err := d.Flush(ctx)
	if err != nil {
		return err
	}

	utils.L(ctx).Normalf("categorizing hashes by last blob")

	hashesByBlob := map[string]map[string]*manifest.Stream{}

	for hash, stream := range d.existing {
		blob := ""
		if len(stream.Ranges) > 0 {
			blob = stream.Ranges[len(stream.Ranges)-1].Blob()
		}
		if _, exists := hashesByBlob[blob]; !exists {
			hashesByBlob[blob] = map[string]*manifest.Stream{}
		}
		hashesByBlob[blob][hash] = stream
	}

	utils.L(ctx).Normalf("found %d in-use last blobs, writing new hash sets",
		len(hashesByBlob))
	if len(hashesByBlob[""]) > 0 {
		utils.L(ctx).Normalf("found %d zero-length hashes", len(hashesByBlob[""]))
	}

	newPaths := make([]string, 0, len(hashesByBlob))
	newSources := map[string]string{}
	for _, hashset := range hashesByBlob {
		path, err := d.flush(ctx, hashset)
		if err != nil {
			return err
		}
		utils.L(ctx).Normalf("wrote new hashset with %d hashes", len(hashset))
		newPaths = append(newPaths, path)
		for hash := range hashset {
			newSources[hash] = path
		}
	}

	utils.L(ctx).Normalf("deleting %d old hashsets...", len(d.paths))
	deleted := map[string]bool{}
	for _, oldpath := range d.paths {
		if deleted[oldpath] {
			continue
		}
		err = d.backend.Delete(ctx, oldpath)
		if err != nil {
			return err
		}
		deleted[oldpath] = true
	}

	d.paths = newPaths
	d.source = newSources

	return nil
}
