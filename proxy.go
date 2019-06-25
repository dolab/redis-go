package redis

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"

	"github.com/segmentio/objconv/resp"
)

// ReverseProxy is the implementation of a redis reverse proxy.
type ReverseProxy struct {
	// Transport specifies the mechanism by which individual requests are made.
	// If nil, DefaultTransport is used.
	Transport RoundTripper

	// The registry exposing the set of redis servers that the proxy routes
	// requests to.
	Registry ServerRegistry

	// ErrorLog specifies an optional logger for errors accepting connections
	// and unexpected behavior from handlers. If nil, logging goes to os.Stderr
	// via the log package's standard logger.
	ErrorLog Logger
}

// ServeRedis satisfies the Handler interface.
func (proxy *ReverseProxy) ServeRedis(w ResponseWriter, r *Request) {
	proxy.serveRequest(w, r)
}

func (proxy *ReverseProxy) serveRequest(w ResponseWriter, req *Request) {
	cmds := req.Cmds
	keys := make([]string, 0, 10)

	for i := range cmds {
		keys = cmds[i].getKeys(keys)
	}

	// TODO: looking up servers and rebuilding the hash ring for every request
	// is not efficient, we should cache and reuse the state.
	ring, err := proxy.lookupServers(req.Context)
	if err != nil {
		proxy.log(err)

		w.Write(errorf("ERR No upstream server was found for the request."))
		return
	}

	upstream := ""
	for _, key := range keys {
		endpoint := ring.LookupServer(key)

		if len(upstream) == 0 {
			upstream = endpoint.Addr
		} else if upstream != endpoint.Addr {
			w.Write(errorf("EXECABORT The transaction contains keys that hash to different upstream servers."))
			return
		}
	}

	req.Addr = upstream

	res, err := proxy.roundTrip(req)
	switch err.(type) {
	case nil:
	case *resp.Error:
		w.Write(err)
		return
	default:
		proxy.log(err)

		proxy.blacklistServer(upstream)

		w.Write(errorf("ERR Connecting to the upstream (%s) server failed.", upstream))
		return
	}

	if res.Args != nil {
		err = proxy.writeArgs(w, res.Args)
	} else {
		err = proxy.writeTxArgs(w, res.TxArgs)
	}

	if err != nil {
		// Get caught by the server, that way the connection is closed and not
		// left in an unpredictable state.
		panic(err)
	}
}

func (proxy *ReverseProxy) writeTxArgs(w ResponseWriter, tx TxArgs) (err error) {
	w.WriteStream(tx.Len())
	var v []interface{} // TODO: figure out a way to avoid loading values in memory

	for a := tx.Next(); a != nil; a = tx.Next() {
		n := 0
		v = append(v, nil)

		for a.Next(&v[n]) {
			v = append(v, nil)
			n++
		}

		err = a.Close()

		if _, ok := err.(*resp.Error); ok {
			v = append(v, err)
			n++
		}

		w.Write(v[:n])
	}

	if e := tx.Close(); e != nil && err == nil {
		err = e
	}

	return
}

func (proxy *ReverseProxy) writeArgs(w ResponseWriter, a Args) (err error) {
	if a.Len() > 1 {
		w.WriteStream(a.Len())
	}

	var v interface{}
	for a.Next(&v) {
		w.Write(v)
		v = nil
	}

	err = a.Close()

	if e, ok := err.(*resp.Error); ok {
		w.Write(e)
		err = nil
	}

	if f, ok := w.(Flusher); ok {
		err = f.Flush()
	}

	return
}

func (proxy *ReverseProxy) servePubSub(conn net.Conn, rw *bufio.ReadWriter, command string, channels ...string) {
	defer conn.Close()

	// TODO:
	// - select the backend server to subscribe to by hashing the channel
	// - refresh the list of servers periodically so we can rebalance when new servers are added
}

func (proxy *ReverseProxy) lookupServers(ctx context.Context) (ring ServerRing, err error) {
	r := proxy.Registry
	if r == nil {
		err = errors.New("a redis proxy needs a non-nil registry to LookupServer the list of available servers")
		return
	}

	return r.LookupServers(ctx)
}

func (proxy *ReverseProxy) blacklistServer(upstream string) {
	if b, ok := proxy.Registry.(ServerBlacklist); ok {
		b.BlacklistServer(ServerEndpoint{Addr: upstream})
	}
}

func (proxy *ReverseProxy) roundTrip(req *Request) (*Response, error) {
	t := proxy.Transport
	if t == nil {
		t = DefaultTransport
	}
	return t.RoundTrip(req)
}

func (proxy *ReverseProxy) log(err error) {
	switch err {
	case io.EOF, io.ErrUnexpectedEOF, io.ErrClosedPipe:
		// Don't log these errors because they are very common and it doesn't
		// bring any value to know that a client disconnected.
		return
	}

	if proxy.ErrorLog != nil {
		proxy.ErrorLog.Print(err)
	} else {
		log.Print(err)
	}
}

func (proxy *ReverseProxy) transport() RoundTripper {
	if transport := proxy.Transport; transport != nil {
		return transport
	}
	return DefaultTransport
}

func errorf(format string, args ...interface{}) error {
	return resp.NewError(fmt.Sprintf(format, args...))
}
