package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/zeebo/errs"

	"github.com/jtolds/jam/backends"
	"github.com/jtolds/jam/blobs"
	"github.com/jtolds/jam/cache"
	"github.com/jtolds/jam/enc"
	"github.com/jtolds/jam/hashdb"
	"github.com/jtolds/jam/session"
)

var (
	sysFlagBlockSize = sysFlags.Int("enc.block-size", 16*1024,
		"encryption block size")
	sysFlagEncKey = sysFlags.String("enc.key", "",
		"hex-encoded 32 byte encryption key,\n\tor locked key (see jam key new/lock)")
	sysFlagStore = sysFlags.String("store",
		(&url.URL{Scheme: "file", Path: filepath.Join(homeDir(), ".jam", "storage")}).String(),
		("place to store data. currently\n\tsupports:\n" +
			"\t* file://<path>,\n" +
			"\t* storj://<access>/<bucket>/<pre>\n" +
			"\t* s3://<bucket>/<prefix>\n" +
			"\t* sftp://<user>@<host>/<prefix>\n" +
			"\tand can be comma-separated to\n\twrite to many at once"))
	sysFlagBlobSize = sysFlags.Int64("blobs.size", 60*1024*1024,
		"target blob size")
	sysFlagMaxUnflushed = sysFlags.Int("blobs.max-unflushed", 1000,
		"max number of objects to stage\n\tbefore flushing (must fit file\n\tdescriptor limit)")
	sysFlagCache = sysFlags.String("cache",
		(&url.URL{Scheme: "file", Path: filepath.Join(homeDir(), ".jam", "cache")}).String(),
		"where to cache things that are\n\tfrequently read")
	sysFlagCacheEnabled = sysFlags.Bool("cache.enabled", true, "if false, disable caching")
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

func getManager(ctx context.Context) (mgr *session.Manager, backend backends.Backend, hashes *hashdb.DB, close func() error, err error) {
	if *sysFlagEncKey == "" {
		return nil, nil, nil, nil, fmt.Errorf("invalid configuration, no root encryption key specified")
	}

	input := bufio.NewReader(os.Stdin)

	var stores []backends.Backend
	defer func() {
		if err != nil {
			for _, store := range stores {
				store.Close()
			}
		}
	}()
	for _, storeurl := range strings.Split(*sysFlagStore, ",") {
		u, err := url.Parse(storeurl)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		store, err := backends.Create(ctx, u)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		stores = append(stores, store)
	}

	store := stores[0]
	if len(stores) > 1 {
		store = backends.Combine(stores[0], stores[1:]...)
	}
	stores = nil
	defer func() {
		if err != nil {
			store.Close()
		}
	}()

	if *sysFlagCacheEnabled {
		cacheURL, err := url.Parse(*sysFlagCache)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		cacheStore, err := backends.Create(ctx, cacheURL)
		if err != nil {
			return nil, nil, nil, nil, err
		}

		wrappedStore, err := cache.New(ctx, store, cacheStore)
		if err != nil {
			cacheStore.Close()
			return nil, nil, nil, nil, err
		}
		// only set store (cleaned up by defer) if err == nil
		store = wrappedStore
	}

	encKey, err := parseKey(os.Stdout, input, *sysFlagEncKey)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	store = enc.NewEncWrapper(
		enc.NewSecretboxCodec(*sysFlagBlockSize),
		enc.NewHMACKeyGenerator(encKey),
		store,
	)
	hashes, err = hashdb.Open(ctx, store)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	blobs := blobs.NewStore(store, *sysFlagBlobSize, *sysFlagMaxUnflushed)
	return session.NewManager(store, blobs, hashes), store, hashes,
		func() error {
			return errs.Combine(blobs.Close(), hashes.Close(), store.Close())
		}, nil
}

func getReadSnapshot(ctx context.Context, mgr *session.Manager, snapshotFlag string) (*session.Snapshot, time.Time, error) {
	if snapshotFlag == "" || snapshotFlag == "latest" {
		return mgr.LatestSnapshot(ctx)
	}
	nano, err := strconv.ParseInt(snapshotFlag, 10, 64)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("invalid snapshot value: %q", snapshotFlag)
	}
	ts := time.Unix(0, nano)
	snap, err := mgr.OpenSnapshot(ctx, ts)
	return snap, ts, err
}
