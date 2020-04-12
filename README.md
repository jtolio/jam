# jam

jam preserves your data

```
USAGE
  jam [opts] <subcommand> [opts]

SUBCOMMANDS
  ls      ls lists files in the given snapshot
  mount   mounts snap as read-only filesystem
  rename  rename allows a regexp-based search and replace against all paths in
          the system, forked from the latest snapshot. See
          https://golang.org/pkg/regexp/#Regexp.ReplaceAllString for semantics.
  snaps   lists snapshots
  store   store adds the given source directory to a new snapshot, forked from
          the latest snapshot.
  unsnap  unsnap removes an old snap

FLAGS
  -blobs.max-unflushed 1000                  max number of objects to stage
                                             before flushing (must fit file
                                             descriptor limit)
  -blobs.size 62914560                       target blob size
  -cache.min-hits 5                          minimum number of hits to a blob
                                             before considering it for caching
  -cache.size 10                             how many blobs to cache
  -cache.store file:///home/user/.jam/cache  where to cache blobs that are
                                             frequently read
  -config /home/user/.jam/jam.conf           path to config file
  -enc.block-size 16384                      encryption block size
  -enc.passphrase ...                        encryption passphrase
  -store file:///home/user/.jam/storage      place to store data. currently
                                             supports:
                                             * file://<path>,
                                             * storj://<access>/<bucket>/<pre>
                                             * s3://<bucket>/<prefix>
                                             and can be comma-separated to
                                             write to many at once
