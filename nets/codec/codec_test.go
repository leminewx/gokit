package codec

import (
	"io"
	"net"
	"time"
)

// fakeConn 模拟 net.Conn，使用 pipe 实现
type fakeConn struct {
	net.Conn
	reader io.Reader
	writer io.Writer
	closed bool
}

func newFakeConn() (*fakeConn, *fakeConn) {
	r1, w1 := io.Pipe()
	r2, w2 := io.Pipe()
	return &fakeConn{reader: r1, writer: w2}, &fakeConn{reader: r2, writer: w1}
}

func (c *fakeConn) Read(b []byte) (int, error)         { return c.reader.Read(b) }
func (c *fakeConn) Write(b []byte) (int, error)        { return c.writer.Write(b) }
func (c *fakeConn) Close() error                       { c.closed = true; return nil }
func (c *fakeConn) LocalAddr() net.Addr                { return &net.TCPAddr{} }
func (c *fakeConn) RemoteAddr() net.Addr               { return &net.TCPAddr{} }
func (c *fakeConn) SetDeadline(t time.Time) error      { return nil }
func (c *fakeConn) SetReadDeadline(t time.Time) error  { return nil }
func (c *fakeConn) SetWriteDeadline(t time.Time) error { return nil }
