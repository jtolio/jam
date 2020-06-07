package sftp

import (
	"context"
	"errors"
	"io"
	"net"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/pkg/sftp"
	"github.com/zeebo/errs"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/knownhosts"

	"github.com/jtolds/jam/backends"
	"github.com/jtolds/jam/blobs"
)

func init() {
	backends.Register("sftp", New)
}

// SFTP implements the Backend interface using SFTP
type SFTP struct {
	root    string
	client  *sftp.Client
	closers []io.Closer
}

// New returns an SFTP backend mounted at the provided path.
func New(ctx context.Context, u *url.URL) (backends.Backend, error) {
	var cfg ssh.ClientConfig
	if u.User != nil {
		cfg.User = u.User.Username()
		if password, ok := u.User.Password(); ok {
			cfg.Auth = append(cfg.Auth, ssh.Password(password))
		}
	}

	us, err := user.Current()
	if err != nil {
		return nil, errs.Wrap(err)
	}
	kh, err := knownhosts.New(filepath.Join(us.HomeDir, ".ssh/known_hosts"))
	if err != nil {
		return nil, errs.Wrap(err)
	}
	cfg.HostKeyCallback = kh

	var closers []io.Closer
	if len(cfg.Auth) == 0 {
		agentConn, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK"))
		if err != nil {
			return nil, errs.Wrap(err)
		}
		closers = append(closers, agentConn)
		cfg.Auth = append(cfg.Auth,
			ssh.PublicKeysCallback(agent.NewClient(agentConn).Signers))
	}

	host := u.Host
	if _, _, err := net.SplitHostPort(host); err != nil {
		if aerr, ok := err.(*net.AddrError); !ok || aerr.Err != "missing port in address" {
			return nil, errs.Wrap(err)
		}
		host += ":22"
	}

	conn, err := ssh.Dial("tcp", host, &cfg)
	if err != nil {
		for _, c := range closers {
			c.Close()
		}
		return nil, errs.Wrap(err)
	}
	closers = append(closers, conn)

	client, err := sftp.NewClient(conn, sftp.UseFstat(true))
	if err != nil {
		for _, c := range closers {
			c.Close()
		}
		return nil, errs.Wrap(err)
	}
	closers = append(closers, client)

	return &SFTP{root: u.Path, client: client, closers: closers}, nil
}

func isNotExist(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, sftp.ErrSSHFxNoSuchFile) {
		return true
	}
	if reflect.DeepEqual(err, errors.New("file does not exist")) {
		return true
	}
	return false
}

// Get implements the Backend interface
func (fs *SFTP) Get(ctx context.Context, path string, offset, length int64) (rv io.ReadCloser, err error) {
	remotepath := filepath.Join(fs.root, path)
	fh, err := fs.client.Open(remotepath)
	if err != nil {
		if isNotExist(err) {
			return nil, errs.Wrap(backends.ErrNotExist)
		}
		return nil, errs.Wrap(err)
	}
	if offset > 0 {
		_, err = fh.Seek(offset, io.SeekStart)
		if err != nil {
			fh.Close()
			return nil, errs.Wrap(err)
		}
	}

	return fh, nil
}

// Put implements the Backend interface
func (fs *SFTP) Put(ctx context.Context, path string, data io.Reader) (err error) {
	defer func() {
		if err != nil {
			fs.Delete(ctx, path)
		}
	}()

	remotepath := filepath.Join(fs.root, path)
	err = fs.client.MkdirAll(filepath.Dir(remotepath))
	if err != nil {
		return errs.Wrap(err)
	}

	tmpfile := filepath.Join(filepath.Dir(remotepath), "_"+blobs.IdGen())
	fh, err := fs.client.OpenFile(tmpfile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC)
	if err != nil {
		return errs.Wrap(err)
	}
	defer fh.Close()

	defer func() {
		if err != nil {
			_ = fs.client.Remove(tmpfile)
		}
	}()

	_, err = io.Copy(fh, data)
	if err != nil {
		return errs.Wrap(err)
	}

	// TODO: use fsync@openssh.com extension, in case the remote is ext4 or similar

	err = fh.Close()
	if err != nil {
		return errs.Wrap(err)
	}

	return errs.Wrap(fs.client.PosixRename(tmpfile, remotepath))
}

// Delete implements the Backend interface
func (fs *SFTP) Delete(ctx context.Context, path string) error {
	remotepath := filepath.Join(fs.root, path)
	if _, err := fs.client.Lstat(remotepath); isNotExist(err) {
		return nil
	}
	err := fs.client.Remove(remotepath)
	if err != nil {
		_, err := fs.client.Lstat(remotepath)
		return errs.New("error: %#v", err)
		return errs.Wrap(err)
	}
	// the rest is not required but is an attempt to be nice and clean up intermediate
	// directories after ourselves. remove any parents up to the root that are empty
	for {
		remotepath = filepath.Dir(remotepath)
		rel, err := filepath.Rel(fs.root, remotepath)
		if err != nil || rel == "." {
			return nil
		}
		err = fs.client.Remove(remotepath)
		if err != nil {
			return nil
		}
	}
}

// List implements the Backend interface
func (fs *SFTP) List(ctx context.Context, prefix string,
	cb func(ctx context.Context, path string) error) error {
	remotepath := filepath.Join(fs.root, prefix)
	if s, err := fs.client.Lstat(remotepath); isNotExist(err) || !s.IsDir() {
		return nil
	}
	walker := fs.client.Walk(filepath.Join(fs.root, prefix))
	for walker.Step() {
		if err := walker.Err(); err != nil {
			return errs.Wrap(err)
		}
		path := walker.Path()
		info := walker.Stat()
		if !info.Mode().IsRegular() ||
			strings.HasPrefix(filepath.Base(path), "_") {
			continue
		}
		internal, err := filepath.Rel(fs.root, path)
		if err != nil {
			return errs.Wrap(err)
		}

		err = cb(ctx, internal)
		if err != nil {
			return err
		}
	}

	return nil
}

func (fs *SFTP) Close() error {
	var eg errs.Group
	for i := len(fs.closers) - 1; i >= 0; i-- {
		eg.Add(fs.closers[i].Close())
	}
	fs.closers = nil
	return eg.Err()
}
