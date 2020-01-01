package protocol

import (
	"net"
	"sync/atomic"
)

type closeConn struct {
	net.Conn
	hasClosed int32
}

func (c *closeConn) Close() error {
	atomic.StoreInt32(&c.hasClosed, 1)
	return c.Conn.Close()
}

func (c *closeConn) HasClosed() bool {
	return atomic.LoadInt32(&c.hasClosed) == 1
}

var _ net.Conn = (*closeConn)(nil)

func netPipe() (*closeConn, *closeConn) {
	conn1, conn2 := net.Pipe()
	return &closeConn{Conn: conn1}, &closeConn{Conn: conn2}
}
