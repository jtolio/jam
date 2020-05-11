# jam

jam preserves your data. you almost certainly want [restic](https://restic.net/)
instead.

```
USAGE
  jam [opts] <subcommand> [opts]

SUBCOMMANDS
  ls         ls lists files in the given snapshot
  mount      mounts snap as read-only filesystem
  rename     rename allows a regexp-based search and replace against all paths in
             the system, forked from the latest snapshot. See
             https://golang.org/pkg/regexp/#Regexp.ReplaceAll for semantics.
  revert-to  revert-to makes a new snapshot that matches an older one
  rm         rm deletes all paths that match the provided regexp.
             https://golang.org/pkg/regexp/#Regexp.Match for semantics.
  snaps      lists snapshots
  store      store adds the given source directory to a new snapshot, forked from
             the latest snapshot.
  unsnap     unsnap removes an old snap
  utils      miscellaneous utilities

FLAGS
  -blobs.max-unflushed 1000            max number of objects to stage
                                       before flushing (must fit file
                                       descriptor limit)
  -blobs.size 62914560                 target blob size
  -cache file:///home/jt/.jam/cache    where to cache blobs that are
                                       frequently read
  -cache.min-hits 5                    minimum number of hits to a blob
                                       before considering it for caching
  -cache.size 10                       how many blobs to cache
  -config /home/jt/.jam/jam.conf       path to config file
  -enc.block-size 16384                encryption block size
  -enc.passphrase ...                  encryption passphrase
  -store file:///home/jt/.jam/storage  place to store data. currently
                                       supports:
                                       * file://<path>,
                                       * storj://<access>/<bucket>/<pre>
                                       * s3://<bucket>/<prefix>
                                       and can be comma-separated to
                                       write to many at once
```
