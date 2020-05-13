// +build !linux

package storj

import (
	"context"
	"net"

	"storj.io/uplink"
)

func newDialer(ctx context.Context) uplink.Transport {
	return &net.Dialer{}
}
