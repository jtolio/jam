// +build !linux

package storj

import (
	"context"
	"net"
)

func newDialer(ctx context.Context) *net.Dialer {
	return &net.Dialer{}
}
