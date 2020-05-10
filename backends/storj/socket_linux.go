package storj

import (
	"context"
	"net"
	"syscall"

	"github.com/jtolds/jam/utils"
)

const af13 = 0x38
const cs1 = 0x8

func newDialer(ctx context.Context) *net.Dialer {
	return &net.Dialer{
		Control: func(network, address string, c syscall.RawConn) error {
			err := c.Control(func(fd uintptr) {
				err := syscall.SetsockoptString(
					int(fd), syscall.IPPROTO_TCP, syscall.TCP_CONGESTION, "ledbat")
				if err != nil {
					utils.L(ctx).Debugf("failed to set congestion controller: %v", err)
				}
				err = syscall.SetsockoptByte(
					int(fd), syscall.SOL_IP, syscall.IP_TOS, cs1)
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
