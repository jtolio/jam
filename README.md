# jam

jam preserves your data

```
USAGE
  ./jam [opts] <subcommand> [opts]

SUBCOMMANDS
  ls      ls lists files in the given snapshot
  mount   mounts snap as read-only filesystem
  rename  rename allows a regexp-based search and replace against all paths in
          the system, forked from the latest snapshot. See
          https://golang.org/pkg/regexp/#Regexp.ReplaceAllString for semantics.
  snaps   lists snapshots
  store   store adds the given source directory to a new snapshot, forked from
          the latest snapshot.

FLAGS
  -blobs.max-unflushed 1000                max number of objects to stage
                                           before flushing (requires file
                                           descriptor limit)
  -blobs.size 62914560                     target blob size
  -config /home/user/.jam/jam.conf         path to config file
  -enc.block-size 16384                    encryption block size
  -enc.passphrase ...                      encryption passphrase
  -store file:///home/user/.jam/storage    place to store data. currently
                                           supports:
                                           * file://<path>,
                                           * storj://<access>/<bucket>/<prefix>
                                           * s3://<bucket>/<prefix>
```
