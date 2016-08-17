package redis

import (
	"bufio"
	"net"
)

// Conn is an interface representing redis connections. It's a simple extension
// to the standard net.Conn interface that adds a couple of methods related to
// buffering, reading and writeing redis requests and responses.
type Conn interface {
	net.Conn

	// Flush writes all buffered data to the underlying connection.
	Flush() error

	// ReadRequest reads a request from the underlying connection.
	ReadRequest() (*Request, error)

	// ReadResponse reads a response from the underlying connection.
	ReadResponse(*Request) (*Response, error)

	// WriteRequest writes a request to the underlying connection.
	WriteRequest(*Request) error

	// WriteResponse writes a response to the underlying connection.
	WriteResponse(*Response) error
}

// NewConn wraps a net.Conn and returns a value that implements Conn.
func NewConn(c net.Conn) Conn {
	return conn{
		Conn: c,
		r:    bufio.NewReaderSize(c, 1024),
		w:    bufio.NewWriterSize(c, 1024),
	}
}

// Dial connects to a redis server at the given address.
func Dial(network string, address string) (conn Conn, err error) {
	var c net.Conn
	if c, err = net.Dial(network, address); err == nil {
		conn = NewConn(c)
	}
	return
}

type conn struct {
	net.Conn
	r *bufio.Reader
	w *bufio.Writer
}

func (c conn) Read(b []byte) (int, error) { return c.r.Read(b) }

func (c conn) Write(b []byte) (int, error) { return c.w.Write(b) }

func (c conn) Flush() error { return c.w.Flush() }

func (c conn) ReadRequest() (req *Request, err error) { return ReadRequest(c.r) }

func (c conn) ReadResponse(req *Request) (res *Response, err error) { return ReadResponse(c.r, req) }

func (c conn) WriteRequest(req *Request) (err error) {
	if err = req.Write(c.w); err == nil {
		err = c.w.Flush()
	}
	return
}

func (c conn) WriteResponse(res *Response) (err error) {
	if err = res.Write(c.w); err == nil {
		err = c.w.Flush()
	}
	return
}
