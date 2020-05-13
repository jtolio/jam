package storj

import (
	"context"
	"net"
	"syscall"

	"github.com/jtolds/jam/utils"
	"storj.io/uplink"
)

const af13 = 0x38

func newDialer(ctx context.Context) uplink.Transport {
	return &net.Dialer{
		Control: func(network, address string, c syscall.RawConn) error {
			err := c.Control(func(fd uintptr) {
				err := syscall.SetsockoptString(
					int(fd), syscall.IPPROTO_TCP, syscall.TCP_CONGESTION, "vegas")
				if err != nil {
					utils.L(ctx).Debugf("failed to set congestion controller: %v", err)
				}
				err = syscall.SetsockoptByte(
					int(fd), syscall.SOL_IP, syscall.IP_TOS, af13)
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
			return nil
		},
	}
}
