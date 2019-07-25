// +build !race

package redis_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dolab/objconv/resp"
	"github.com/golib/assert"
	"github.com/google/uuid"

	"github.com/dolab/redis-go"
	"github.com/dolab/redis-go/redistest"
)

func TestServerMetrics(t *testing.T) {
	it := assert.New(t)

	respErr := resp.NewError("ERR something went wrong")

	var counter int64
	srv, addr := redistest.FakeMetricsServer(redis.HandlerFunc(func(res redis.ResponseWriter, req *redis.Request) {
		if atomic.AddInt64(&counter, 1)%2 == 0 {
			res.Write(respErr)
		} else {
			res.Write("OK")
		}

	}), time.Second)
	defer srv.Close()

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

	// for connection
	it.Contains(output, `redis_proxy_connections{local_addr="127.0.0.1",remote_addr="127.0.0.1"}`)
	it.Contains(output, `redis_proxy_reconnects_total{local_addr="127.0.0.1",remote_addr="127.0.0.1"}`)

	// for request
	it.Contains(output, `redis_proxy_requests_total{local_addr="127.0.0.1",remote_addr="127.0.0.1"}`)
	it.Contains(output, `redis_proxy_requests{local_addr="127.0.0.1",remote_addr="127.0.0.1"}`)

	// for commands
	it.NotContains(output, `redis_proxy_commands_total{cmd="PING",remote_addr="127.0.0.1"}`)
	it.Contains(output, `redis_proxy_commands_total{cmd="SET",local_addr="127.0.0.1",remote_addr="127.0.0.1"}`)
	it.Contains(output, `redis_proxy_commands_total{cmd="GET",local_addr="127.0.0.1",remote_addr="127.0.0.1"}`)
	it.Contains(output, `redis_proxy_commands_total{cmd="DEL",local_addr="127.0.0.1",remote_addr="127.0.0.1"}`)
	it.NotContains(output, `redis_proxy_commands{cmd="PING",remote_addr="127.0.0.1"}`)
	it.Contains(output, `redis_proxy_commands{cmd="SET",local_addr="127.0.0.1",remote_addr="127.0.0.1"}`)
	it.Contains(output, `redis_proxy_commands{cmd="GET",local_addr="127.0.0.1",remote_addr="127.0.0.1"}`)
	it.Contains(output, `redis_proxy_commands{cmd="DEL",local_addr="127.0.0.1",remote_addr="127.0.0.1"}`)

	// for duration
	it.Contains(output, `redis_proxy_redis_duration_seconds_sum{local_addr="127.0.0.1",remote_addr="127.0.0.1"}`)
	it.Contains(output, `redis_proxy_redis_duration_seconds_count{local_addr="127.0.0.1",remote_addr="127.0.0.1"}`)
}
