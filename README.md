# jam

jam preserves your data. you almost certainly want [restic](https://restic.net/)
instead.

```
DESCRIPTION
  jam preserves your data

USAGE
  jam [opts] <subcommand> [opts]

SUBCOMMANDS
  integrity  integrity check. for full effect, disable caching and enable read
             comparison
  key        encryption key utilities
  ls         ls lists files in the given snapshot
  mount      mounts snap as read-only filesystem
  rename     rename allows a regexp-based search and replace against all paths
             in the system, forked from the latest snapshot. See
             https://golang.org/pkg/regexp/#Regexp.ReplaceAll for semantics.
  revert-to  revert-to makes a new snapshot that matches an older one
  rm         rm deletes all paths that match the provided prefix
  snaps      lists snapshots
  store      store adds the given source directory to a new snapshot, forked
             from the latest snapshot.
  unsnap     unsnap removes an old snap
  utils      miscellaneous utilities
  webdav     serves snap as read-only webdav

FLAGS
  -blobs.max-unflushed 1000            max number of objects to stage
                                       before flushing (must fit file
                                       descriptor limit)
  -blobs.size 62914560                 target blob size
  -cache file:///home/jt/.jam/cache    where to cache things that are
                                       frequently read
  -cache.blobs=false                   if true and caching is enabled, cache blobs
  -cache.enabled=true                  if false, disable caching
  -config /home/jt/.jam/jam.conf       path to config file
  -enc.block-size 16384                default encryption block size
  -enc.block-size-small 1024           encryption block size for small objects
  -enc.key string                      hex-encoded 32 byte encryption key,
                                       or locked key (see jam key new/lock)
  -log.level normal                    default log level. can be:
                                       debug, normal, urgent, or none
  -store file:///home/jt/.jam/storage  place to store data. currently
                                       supports:
                                       * file://<path>,
                                       * storj://<access>/<bucket>/<pre>
                                       * s3://<ak>:<sk>@<region>/<bkt>/<pre>
                                       * sftp://<user>@<host>/<prefix>
                                       and can be comma-separated to
                                       write to many at once
  -store.read-compare=false            if true, compare reads across
                                       all backends. useful for integrity
                                       checking
```
