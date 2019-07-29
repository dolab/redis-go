package redis

import (
	"context"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golib/assert"
)

var (
	fakeServer = func(tag string, handler Handler, pipe bool) *Server {
		return &Server{
			Addr:           ":6379",
			Handler:        handler,
			EnableRetry:    false,
			EnablePipeline: pipe,
			ErrorLog:       log.New(os.Stdout, "["+tag+"]", os.O_CREATE|os.O_WRONLY|os.O_APPEND),
		}
	}
)

func Test_serveConnectionWithoutPipeline(t *testing.T) {
	it := assert.New(t)

	cmds := []byte("*5\r\n$3\r\nset\r\n$36\r\nfccf5abb-dd2d-45c4-853f-62e36ae60cbc\r\n$36\r\neb405cad-cae2-4af9-b2c8-e81fd0bc154a\r\n$2\r\nex\r\n$1\r\n1\r\n*2\r\n$3\r\nget\r\n$36\r\nfccf5abb-dd2d-45c4-853f-62e36ae60cbc\r\n")

	var (
		cmdCases = map[int64]string{
			0: "set",
			1: "get",
		}
		cmdIndex int64
		cmdWg    sync.WaitGroup
	)
	srv := fakeServer("server without pipeline", HandlerFunc(func(w ResponseWriter, r *Request) {
		it.Equal(1, len(r.Cmds))
		it.Equal(cmdCases[cmdIndex], r.Cmds[0].Cmd)

		atomic.AddInt64(&cmdIndex, 1)

		w.Write("OK")
	}), false)

	reader, writer := net.Pipe()

	cmdWg.Add(1)
	go func() {
		defer cmdWg.Done()

		n, err := writer.Write(cmds)
		if it.Nil(err) {
			it.Equal(len(cmds), n)
		}

		b, err := ioutil.ReadAll(writer)
		if it.Nil(err) {
			it.Equal("+OK\r\n+OK\r\n", string(b))
		}

		it.Nil(writer.Close())
	}()

	ctx := context.Background()
	conn := NewServerConn(reader)

	srv.serveConnection(ctx, conn, serverConfig{
		idleTimeout:  time.Millisecond,
		readTimeout:  time.Millisecond,
		writeTimeout: time.Millisecond,
		retryable:    false,
	})

	it.Nil(reader.Close())
	it.EqualValues(2, cmdIndex)

	cmdWg.Wait()
}

func Benchmark_serveConnectionWithoutPipeline(b *testing.B) {
	logger := log.New(os.Stdout, "[serve connection pipeline]", os.O_CREATE|os.O_WRONLY|os.O_APPEND)
	cmds := []byte("*5\r\n$3\r\nset\r\n$36\r\nfccf5abb-dd2d-45c4-853f-62e36ae60cbc\r\n$36\r\neb405cad-cae2-4af9-b2c8-e81fd0bc154a\r\n$2\r\nex\r\n$1\r\n1\r\n*2\r\n$3\r\nget\r\n$36\r\nfccf5abb-dd2d-45c4-853f-62e36ae60cbc\r\n")

	srv := &Server{
		Addr: ":6379",
		Handler: HandlerFunc(func(w ResponseWriter, r *Request) {
			w.Write("OK")
		}),
		EnableRetry:    false,
		EnablePipeline: false,
		ErrorLog:       logger,
	}

	reader, writer := net.Pipe()
	go func() {
		for {
			if _, err := writer.Write(cmds); err != nil {
				log.Println(err.Error())
				return
			}

			if _, err := io.Copy(ioutil.Discard, writer); err != nil {
				log.Println(err.Error())
				return
			}
		}
	}()

	ctx := context.Background()
	conn := NewServerConn(reader)

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			srv.serveConnection(ctx, conn, serverConfig{
				idleTimeout:  time.Millisecond,
				readTimeout:  time.Millisecond,
				writeTimeout: time.Millisecond,
				retryable:    false,
			})
		}
	})
}

func Test_serveConnectionWithPipeline(t *testing.T) {
	it := assert.New(t)

	cmds := []byte("*5\r\n$3\r\nset\r\n$36\r\nfccf5abb-dd2d-45c4-853f-62e36ae60cbc\r\n$36\r\neb405cad-cae2-4af9-b2c8-e81fd0bc154a\r\n$2\r\nex\r\n$1\r\n1\r\n*2\r\n$3\r\nget\r\n$36\r\nfccf5abb-dd2d-45c4-853f-62e36ae60cbc\r\n")

	var (
		cmdCases = map[int64]string{
			0: "set",
			1: "get",
		}
		cmdIndex int64
		cmdWg    sync.WaitGroup
	)

	srv := fakeServer("server with pipeline", HandlerFunc(func(w ResponseWriter, r *Request) {
		if it.Equal(2, len(r.Cmds)) {
			it.Equal(cmdCases[0], r.Cmds[0].Cmd)
			it.Equal(cmdCases[1], r.Cmds[1].Cmd)
		}

		atomic.AddInt64(&cmdIndex, 1)

		w.Write("OK")
	}), true)

	reader, writer := net.Pipe()

	cmdWg.Add(1)
	go func() {
		defer cmdWg.Done()

		n, err := writer.Write(cmds)
		if it.Nil(err) {
			it.Equal(len(cmds), n)
		}

		b, err := ioutil.ReadAll(writer)
		if it.Nil(err) {
			it.Equal("+OK\r\n", string(b))
		}

		it.Nil(writer.Close())
	}()

	ctx := context.Background()
	conn := NewServerConn(reader)

	srv.serveConnection(ctx, conn, serverConfig{
		idleTimeout:  time.Millisecond,
		readTimeout:  time.Millisecond,
		writeTimeout: time.Millisecond,
		retryable:    false,
	})

	it.Nil(reader.Close())
	it.EqualValues(1, cmdIndex)

	cmdWg.Wait()
}

func Benchmark_serveConnectionWithPipeline(b *testing.B) {
	logger := log.New(os.Stdout, "[serve connection pipeline]", os.O_CREATE|os.O_WRONLY|os.O_APPEND)
	cmds := []byte("*5\r\n$3\r\nset\r\n$36\r\nfccf5abb-dd2d-45c4-853f-62e36ae60cbc\r\n$36\r\neb405cad-cae2-4af9-b2c8-e81fd0bc154a\r\n$2\r\nex\r\n$1\r\n1\r\n*2\r\n$3\r\nget\r\n$36\r\nfccf5abb-dd2d-45c4-853f-62e36ae60cbc\r\n")

	srv := &Server{
		Addr: ":6379",
		Handler: HandlerFunc(func(w ResponseWriter, r *Request) {
			w.Write("OK")
		}),
		EnableRetry:    false,
		EnablePipeline: true,
		ErrorLog:       logger,
	}

	reader, writer := net.Pipe()
	go func() {
		for {
			if _, err := writer.Write(cmds); err != nil {
				log.Println(err.Error())
				return
			}

			if _, err := io.Copy(ioutil.Discard, writer); err != nil {
				log.Println(err.Error())
				return
			}
		}
	}()

	ctx := context.Background()
	conn := NewServerConn(reader)

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			srv.serveConnection(ctx, conn, serverConfig{
				idleTimeout:  time.Millisecond,
				readTimeout:  time.Millisecond,
				writeTimeout: time.Millisecond,
				retryable:    false,
			})
		}
	})
}
