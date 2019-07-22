// +build !race

package redis_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/dolab/objconv/resp"
	"github.com/dolab/redis-go"
	"github.com/dolab/redis-go/redistest"
	"github.com/golib/assert"
	"github.com/google/uuid"
)

func TestServerMetrics(t *testing.T) {
	it := assert.New(t)

	respErr := resp.NewError("ERR something went wrong")

	var counter int64
	srv, addr := redistest.FakeServer(redis.HandlerFunc(func(res redis.ResponseWriter, req *redis.Request) {
		if atomic.AddInt64(&counter, 1)%2 == 0 {
			res.Write(respErr)
		} else {
			res.Write("OK")
		}

	}))
	defer srv.Close()

	srv.WithMetrics()

	tr := &redis.Transport{MaxIdleConns: 1}
	defer tr.CloseIdleConnections()

	cli := &redis.Client{Addr: addr, Transport: tr}
	key := uuid.New().String()
	value := uuid.New().String()

	// set
	setErr := cli.Exec(context.Background(), "SET", key, value)
	if it.Nil(setErr) {
		// del, it should return error
		delErr := cli.Exec(context.Background(), "DEL", key)
		it.EqualErrors(respErr, delErr)

		// get, it should ok
		getCmd := cli.Query(context.Background(), "GET", key)

		var getValue string
		if getCmd.Next(&getValue) {
			it.Equal("OK", getValue)
		}
	}

	// for gometrics
	r := httptest.NewRequest(http.MethodGet, "https://example.com", nil)
	w := httptest.NewRecorder()

	redis.ServeMetrics(w, r)

	output := w.Body.String()
	it.Contains(output, `redis_server_requests_total{remote_addr="127.0.0.1"}`)
	it.Contains(output, `redis_server_requests{remote_addr="127.0.0.1"}`)
	it.NotContains(output, `redis_server_commands_total{cmd="PING",remote_addr="127.0.0.1"}`)
	it.Contains(output, `redis_server_commands_total{cmd="SET",remote_addr="127.0.0.1"}`)
	it.Contains(output, `redis_server_commands_total{cmd="GET",remote_addr="127.0.0.1"}`)
	it.Contains(output, `redis_server_commands_total{cmd="DEL",remote_addr="127.0.0.1"}`)
	it.NotContains(output, `redis_server_commands{cmd="PING",remote_addr="127.0.0.1"}`)
	it.Contains(output, `redis_server_commands{cmd="SET",remote_addr="127.0.0.1"}`)
	it.Contains(output, `redis_server_commands{cmd="GET",remote_addr="127.0.0.1"}`)
	it.Contains(output, `redis_server_commands{cmd="DEL",remote_addr="127.0.0.1"}`)
}
