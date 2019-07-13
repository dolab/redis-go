package redis_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golib/assert"

	"github.com/dolab/redis-go"
	"github.com/dolab/redis-go/redistest"
)

// for client retry
type retryTransport struct {
	*redis.Transport

	assert   *assert.Assertions
	endpoint redis.ServerEndpoint
}

func (retry *retryTransport) RoundTrip(req *redis.Request) (resp *redis.Response, err error) {
	resp, err = retry.Transport.RoundTrip(req)

	if retry.assert.Nil(err) {
		if retry.assert.True(resp.IsRespError()) {

			// retry
			retryReq, retryErr := resp.Retry()
			if retry.assert.Nil(retryErr) {
				retry.assert.Equal(retry.endpoint.Addr, retryReq.Addr)

				newResp, newErr := retry.Transport.RoundTrip(retryReq)

				if retry.assert.Nil(newErr) {
					retry.assert.False(newResp.IsRespError())

					// correct response by overwriting
					resp = newResp
				}
			}
		}
	}

	return
}

func TestServerTransportWithRetry(t *testing.T) {
	it := assert.New(t)

	validServers, _, _ := redistest.FakeServerList()

	ring, _ := validServers.LookupServers(context.Background())
	cmd := "SET"
	key := "6399229a-b5f0-451c-b6f0-2b05f1dd6553"
	value := "0123456789"

	var moved int64
	<-redistest.TestServer(validServers, func(w redis.ResponseWriter, r *redis.Request) {
		n := atomic.AddInt64(&moved, 1)
		if n > 1 {
			if it.Equal(1, len(r.Cmds)) {
				reqCmd := r.Cmds[0]
				it.Equal(cmd, reqCmd.Cmd)

				var (
					reqKey, reqValue string
				)

				perr := redis.ParseArgs(reqCmd.Args, &reqKey, &reqValue)
				if it.Nil(perr) {
					it.Equal(key, reqKey)
					it.Equal(value, reqValue)
				}
			}

			w.Write("OK")
		} else {
			w.Write(fmt.Errorf("MOVED 1 %s", ring.LookupServer(key).Addr))
		}
	})

	transport := &redis.Transport{}
	retryTransport := &retryTransport{
		Transport: transport,
		assert:    it,
		endpoint:  ring.LookupServer(key),
	}
	proxy := &redis.ReverseProxy{
		Transport: retryTransport,
		Registry:  validServers,
		ErrorLog:  log.New(os.Stdout, "[Proxy Testing]", 0),
	}

	_, serverAddr := redistest.FakeTimeoutServer(proxy, 1000*time.Millisecond)
	client := redis.Client{
		Addr:      serverAddr,
		Transport: transport,
		Timeout:   time.Second,
	}

	err := client.Exec(context.Background(), cmd, key, value, "ex", time.Second)
	it.Nil(err)
}
