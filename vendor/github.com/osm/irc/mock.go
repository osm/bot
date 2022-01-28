package irc

import (
	"io"
	"net"
	"time"
)

// addr holds the internal data of the implementation of the net.Addr interface
type addr struct {
	typ  string
	addr string
}

// Methods required by the net.Addr interface
func (a *addr) Network() string { return a.typ }
func (a *addr) String() string  { return a.addr }

// mockConn holds the internal data structures needed to mock a TCP connection
type mockConn struct {
	reader *io.PipeReader
	writer *io.PipeWriter
}

// Methods required to satisfy the net.Conn interface
func (c mockConn) Read(data []byte) (int, error)      { return c.reader.Read(data) }
func (c mockConn) Write(data []byte) (int, error)     { return c.writer.Write(data) }
func (c mockConn) LocalAddr() net.Addr                { return &addr{"tcp", "127.0.0.1"} }
func (c mockConn) RemoteAddr() net.Addr               { return &addr{"tcp", "127.0.0.1"} }
func (c mockConn) SetDeadline(t time.Time) error      { return nil }
func (c mockConn) SetReadDeadline(t time.Time) error  { return nil }
func (c mockConn) SetWriteDeadline(t time.Time) error { return nil }

func (c mockConn) Close() error {
	if err := c.reader.Close(); err != nil {
		return err
	}

	if err := c.writer.Close(); err != nil {
		return err
	}

	return nil
}

// mockComm contains the mocked connections for client -> server and server -> client communication
type mockComm struct {
	Client *mockConn
	Server *mockConn
}

// newMockComm creates a new mocked communications hub
func newMockComm() *mockComm {
	clientRead, serverWrite := io.Pipe()
	serverRead, clientWrite := io.Pipe()

	return &mockComm{
		Client: &mockConn{
			reader: clientRead,
			writer: clientWrite,
		},
		Server: &mockConn{
			reader: serverRead,
			writer: serverWrite,
		},
	}
}
