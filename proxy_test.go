package redis_test

import (
	"context"
	"log"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/golib/assert"
	"github.com/google/uuid"

	"github.com/dolab/redis-go"
	"github.com/dolab/redis-go/redistest"
)

func TestReverseProxy(t *testing.T) {
	redistest.TestClient(t, func() (redistest.Client, func(), error) {
		transport := &redis.Transport{}

		validServers, _, _ := makeServerList()

		<-redistest.TestServer(validServers)

		_, serverAddr := newServer(&redis.ReverseProxy{
			Transport: transport,
			Registry:  validServers,
			ErrorLog:  log.New(os.Stderr, "[Proxy] ==> ", 0),
		})

		teardown := func() {
			transport.CloseIdleConnections()
		}

		return &testClient{Client: redis.Client{Addr: serverAddr, Transport: transport}}, teardown, nil
	})
}

func TestReverseProxyHash(t *testing.T) {
	it := assert.New(t)
	transport := &redis.Transport{}

	validServers, brokenServers, oneDownServers := makeServerList()

	<-redistest.TestServer(validServers)

	proxy := &redis.ReverseProxy{
		Transport: transport,
		Registry:  validServers,
		ErrorLog:  log.New(os.Stderr, "[Proxy Hash]", 0),
	}

	_, serverAddr := newServerTimeout(proxy, 1000*time.Millisecond)
	client := &redis.Client{
		Addr:      serverAddr,
		Transport: transport,
	}

	// validServers backend - write max keys
	var (
		max     = 160
		templ   = "redis-go.hash." + uuid.New().String() + ".%d"
		sleep   = 100             // millisecond
		timeout = 1 * time.Second // client timeout

		numHits, numMisses, numSuccess, numFailure, numErrs int
		err                                                 error
	)
	_, _, _ = numHits, numMisses, numErrs

	numSuccess, numFailure, err = redistest.WriteTestPattern(client, max, templ, sleep, timeout)
	if it.Nil(err) {
		it.Equal(max, numSuccess, "All writes succeeded")
		it.Equal(0, numFailure, "No writes failed")
	}

	// validServers backend - read max back
	numHits, numMisses, numErrs, err = redistest.ReadTestPattern(client, max, templ, sleep, timeout)
	t.Logf("validServers backend max read: numHits = %d, numMisses = %d, numErrs = %d, err = %v", numHits, numMisses, numErrs, err)
	if it.Nil(err, "Full backend - no errors") {
		it.Equal(max, numHits, "Full backend - all hit")
		it.Equal(0, numMisses, "Full backend - none missed")
		it.Equal(0, numErrs, "Full backend - no errors")
	}

	// validServers backend - read max+1 back
	numHits, numMisses, numErrs, err = redistest.ReadTestPattern(client, max+1, templ, sleep, timeout)
	t.Logf("validServers backend max+1 read: numHits = %d, numMisses = %d, numErrs = %d, err = %v", numHits, numMisses, numErrs, err)
	if it.Nil(err, "Extra read - no errors") {
		it.Equal(max, numHits, "Extra read - all hit")
		it.Equal(1, numMisses, "Extra read - one (additional) missed")
		it.Equal(0, numErrs, "Extra read - no errors")
	}

	// malfunctioning backend - read oneDownServers
	proxy.Registry = brokenServers
	numHits, numMisses, numErrs, err = redistest.ReadTestPattern(client, 1, templ, sleep, 2*time.Second)
	t.Logf("brokenServers backend max read: numHits = %d, numMisses = %d, numErrs = %d, err = %v", numHits, numMisses, numErrs, err)
	if it.NotNil(err, "Broken backend - errors") {
		it.Equal(0, numHits, "Broken backend - none hit")
		it.Equal(0, numMisses, "Broken backend - none missed")
		it.Equal(1, numErrs, "Broken backend - all errors")
	}

	// single backend dropped (all combinations) - read max back
	accHits, accMisses := 0, 0
	for i := 0; i < len(oneDownServers); i++ {
		proxy.Registry = oneDownServers[i]

		numHits, numMisses, numErrs, err := redistest.ReadTestPattern(client, max, templ, sleep, timeout)
		t.Logf("single backend dropped (%d): numHits = %d, numMisses = %d, numErrs = %d, %d%% miss rate, err = %v", i, numHits, numMisses, numErrs, 100*numMisses/max, err)

		if it.Nil(err, "One down - no errors") {
			it.True(numHits < max, "One down - not all hit")
			it.True(numHits > 0, "One down - some hit")
			it.True(numMisses < max, "One down - not all missed")
			it.True(numMisses > 0, "One down - some missed")
			it.Equal(max, numHits+numMisses, "One down - hits and misses adds up")
			it.Equal(0, numErrs, "One down - no errors")
		}

		accHits += numHits
		accMisses += numMisses
	}
	it.Equal(max, accMisses, "Misses add up")
}

func TestReverseProxy_ServeRedisWithOneshot(t *testing.T) {
	it := assert.New(t)

	validServers, _, _ := makeServerList()
	<-redistest.TestServer(validServers)

	proxy := &redis.ReverseProxy{
		Transport: &redis.Transport{},
		Registry:  validServers,
		ErrorLog:  log.New(os.Stderr, "[Proxy Hash Oneshot] ==> ", 0),
	}

	request := redis.NewRequest(validServers[0].Addr, "SET", redis.List("key", "value"))
	request.Context = context.TODO()

	response := &responseWriter{}

	proxy.ServeRedis(response, request)

	it.False(response.stream)
	it.Zero(response.shots)
}

var benchmarkReverseProxyOnce sync.Once

func BenchmarkReverseProxy_ServeRedis(b *testing.B) {
	transport := &redis.Transport{}

	validServers, _, _ := makeServerList()

	benchmarkReverseProxyOnce.Do(func() {
		<-redistest.TestServer(validServers)
	})

	proxy := &redis.ReverseProxy{
		Transport: transport,
		Registry:  validServers,
		ErrorLog:  log.New(os.Stderr, "[Proxy Hash Bench] ==> ", 0),
	}

	request := redis.NewRequest(validServers[0].Addr, "SET", redis.List("key", "value"))
	request.Context = context.TODO()

	response := &responseWriter{}

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			proxy.ServeRedis(response, request)
		}
	})
}

type responseWriter struct {
	mux    sync.Mutex
	shots  int
	stream bool
	values []interface{}
}

func (w *responseWriter) WriteStream(n int) error {
	w.mux.Lock()
	w.shots = n
	w.stream = true
	w.mux.Unlock()

	return nil
}

func (w *responseWriter) Write(v interface{}) error {
	w.mux.Lock()
	w.values = append(w.values, v)
	w.mux.Unlock()

	return nil
}
