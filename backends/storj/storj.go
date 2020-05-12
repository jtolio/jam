package storj

import (
	"context"
	"io"
	"net"
	"net/url"
	"strings"
	"syscall"

	"storj.io/uplink"

	"github.com/jtolds/jam/backends"
	"github.com/jtolds/jam/utils"
	"github.com/zeebo/errs"
)

var (
	Error = errs.Class("storj error")
)

func init() {
	backends.Register("storj", New)
}

type Backend struct {
	p      *uplink.Project
	bucket string
	prefix string
}

type dialer struct {
	dialer uplink.Transport
}

func (d dialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	conn, err := d.dialer.DialContext(ctx, network, address)
	if err != nil {
		return nil, err
	}
	raw, err := conn.(interface {
		SyscallConn() (syscall.RawConn, error)
	}).SyscallConn()
	if err != nil {
		return nil, err
	}
	err = raw.Control(func(fd uintptr) {
		err := syscall.SetsockoptString(
			int(fd), syscall.IPPROTO_TCP, syscall.TCP_CONGESTION, "lp")
		if err != nil {
			utils.L(ctx).Debugf("failed to set congestion controller: %v", err)
		}
		err = syscall.SetsockoptByte(
			int(fd), syscall.SOL_IP, syscall.IP_TOS, 0)
		if err != nil {
			utils.L(ctx).Debugf("failed to set ip tos: %v", err)
		}
		err = syscall.SetsockoptInt(
			int(fd), syscall.SOL_SOCKET, syscall.SO_PRIORITY, 0)
		if err != nil {
			utils.L(ctx).Debugf("failed to set socket priority: %v", err)
		}
	})
	if err != nil {
		utils.L(ctx).Debugf("failed to set socket settings: %v", err)
	}
	return conn, nil
}

func New(ctx context.Context, u *url.URL) (backends.Backend, error) {
	access, err := uplink.ParseAccess(u.Host)
	if err != nil {
		return nil, Error.Wrap(err)
	}

	cfg := uplink.Config{Transport: dialer{dialer: &net.Dialer{}}}

	p, err := cfg.OpenProject(ctx, access)
	if err != nil {
		return nil, Error.Wrap(err)
	}
	parts := strings.SplitN(strings.TrimPrefix(u.Path, "/"), "/", 2)
	var prefix string
	if len(parts) > 1 {
		prefix = parts[1]
	}

	return &Backend{
		p:      p,
		bucket: parts[0],
		prefix: prefix,
	}, nil
}

var _ backends.Backend = (*Backend)(nil)

func (b *Backend) Get(ctx context.Context, path string, offset, length int64) (io.ReadCloser, error) {
	path = b.prefix + path
	d, err := b.p.DownloadObject(ctx, b.bucket, path, &uplink.DownloadOptions{Offset: offset, Length: length})
	return d, Error.Wrap(err)
}

func (b *Backend) Put(ctx context.Context, path string, data io.Reader) error {
	path = b.prefix + path
	u, err := b.p.UploadObject(ctx, b.bucket, path, nil)
	if err != nil {
		return Error.Wrap(err)
	}
	defer u.Abort()
	_, err = io.Copy(u, data)
	if err != nil {
		return Error.Wrap(err)
	}
	return Error.Wrap(u.Commit())
}

func (b *Backend) Delete(ctx context.Context, path string) error {
	path = b.prefix + path
	_, err := b.p.DeleteObject(ctx, b.bucket, path)
	return Error.Wrap(err)
}

func (b *Backend) List(ctx context.Context, prefix string, cb func(ctx context.Context, path string) error) error {
	prefix = b.prefix + prefix
	it := b.p.ListObjects(ctx, b.bucket, &uplink.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	})
	for it.Next() {
		err := cb(ctx, strings.TrimPrefix(it.Item().Key, b.prefix))
		if err != nil {
			return err
		}
	}
	return Error.Wrap(it.Err())
}

func (b *Backend) Close() error {
	return Error.Wrap(b.p.Close())
}
