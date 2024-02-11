package quicpunch

import (
	"errors"
	"net"

	"github.com/pion/ice/v2"
)

var _ net.PacketConn = &packetConnAdapter{}

type packetConnAdapter struct {
	*ice.Conn
}

func (c *packetConnAdapter) ReadFrom(p []byte) (n int, addr net.Addr, err error) {
	addr = c.Conn.RemoteAddr()
	n, err = c.Conn.Read(p)
	return
}

func (c *packetConnAdapter) WriteTo(p []byte, addr net.Addr) (n int, err error) {
	if addr != c.Conn.RemoteAddr() {
		return 0, errors.New("invalid target address: doesn't match ICE peer")
	}
	n, err = c.Conn.Write(p)
	return
}
