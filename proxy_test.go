package redis_test

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/url"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/golib/assert"

	"github.com/dolab/redis-go"
	"github.com/dolab/redis-go/redistest"
)

func TestReverseProxy(t *testing.T) {
	redistest.TestClient(t, func() (redistest.Client, func(), error) {
		transport := &redis.Transport{}

		serverList, _, _ := makeServerList()

		_, serverURL := newServer(&redis.ReverseProxy{
			Transport: transport,
			Registry:  serverList,
			ErrorLog:  log.New(os.Stderr, "proxy test ==> ", 0),
		}, serverList)

		teardown := func() {
			transport.CloseIdleConnections()
		}

		u, _ := url.Parse(serverURL)
		return &testClient{Client: redis.Client{Addr: u.Host, Transport: transport}}, teardown, nil
	})
}

func TestReverseProxyHash(t *testing.T) {
	it := assert.New(t)
	transport := &redis.Transport{}

	full, broken, onedowns := makeServerList()
	_ = broken

	proxy := &redis.ReverseProxy{
		Transport: transport,
		Registry:  full,
		ErrorLog:  log.New(os.Stderr, "proxy hash test ==> ", 0),
	}

	_, serverURL := newServerTimeout(proxy, 10000*time.Millisecond, full)
	u, _ := url.Parse(serverURL)
	client := &redis.Client{
		Addr:      u.Host,
		Transport: transport,
	}

	// full backend - write n keys
	n := 160
	keyTempl := "redis-go.test.rphash.%d"
	sleep := 5 * time.Second
	timeout := 30 * time.Second

	numSuccess, numFailure, err := redistest.WriteTestPattern(client, n, keyTempl, sleep, timeout)
	if it.Nil(err) {
		it.Equal(n, numSuccess, "All writes succeeded")
		it.Equal(0, numFailure, "No writes failed")
	}

	// full backend - read n back
	numHits, numMisses, numErrs, err := redistest.ReadTestPattern(client, n, keyTempl, sleep, timeout)
	t.Logf("full backend n read: numHits = %d, numMisses = %d, numErrs = %d, err = %v", numHits, numMisses, numErrs, err)
	if it.Nil(err, "Full backend - no errors") {
		it.Equal(n, numHits, "Full backend - all hit")
		it.Equal(0, numMisses, "Full backend - none missed")
		it.Equal(0, numErrs, "Full backend - no errors")
	}

	// full backend - read n+1 back
	numHits, numMisses, numErrs, err = redistest.ReadTestPattern(client, n+1, keyTempl, sleep, timeout)
	t.Logf("full backend n+1 read: numHits = %d, numMisses = %d, numErrs = %d, err = %v", numHits, numMisses, numErrs, err)
	if it.Nil(err, "Extra read - no errors") {
		it.Equal(n, numHits, "Extra read - all hit")
		it.Equal(1, numMisses, "Extra read - one (additional) missed")
		it.Equal(0, numErrs, "Extra read - no errors")
	}

	// // malfunctioning backend - read fails
	// proxy.Registry = broken
	// numHits, numMisses, numErrs, err = redistest.ReadTestPattern(client, 1, keyTempl, sleep, 2*time.Second)
	// t.Logf("broken backend n read: numHits = %d, numMisses = %d, numErrs = %d, err = %v", numHits, numMisses, numErrs, err)
	// if it.NotNil(err, "Broken backend - errors") {
	// 	it.Equal(0, numHits, "Broken backend - none hit")
	// 	it.Equal(0, numMisses, "Broken backend - none missed")
	// 	it.Equal(1, numErrs, "Broken backend - all errors")
	// }

	// single backend dropped (all combinations) - read n back
	accHits, accMisses := 0, 0
	for i := 0; i < len(onedowns); i++ {
		proxy.Registry = onedowns[i]
		numHits, numMisses, numErrs, err := redistest.ReadTestPattern(client, n, keyTempl, sleep, timeout)
		t.Logf("single backend dropped (%d): numHits = %d, numMisses = %d, numErrs = %d, %d%% miss rate, err = %v", i, numHits, numMisses, numErrs, 100*numMisses/n, err)
		if it.Nil(err, "One down - no errors") {
			it.True(numHits < n, "One down - not all hit")
			it.True(numHits > 0, "One down - some hit")
			it.True(numMisses < n, "One down - not all missed")
			it.True(numMisses > 0, "One down - some missed")
			it.Equal(n, numHits+numMisses, "One down - hits and misses adds up")
			it.Equal(0, numErrs, "One down - no errors")
		}
		accHits += numHits
		accMisses += numMisses
	}
	it.Equal(n, accMisses, "Misses add up")
}

var benchmarkReverseProxyOnce sync.Once

func BenchmarkReverseProxy_ServeRedis(b *testing.B) {
	transport := &redis.Transport{}

	endpoints, _, _ := makeServerList()

	benchmarkReverseProxyOnce.Do(func() {
		for _, endpoint := range endpoints {
			go func(addr string) {
				log.Println("Starting server ", addr)

				log.Fatal(redis.ListenAndServe(addr, redis.HandlerFunc(func(w redis.ResponseWriter, r *redis.Request) {
					w.Write("OK")
					return
				})))
			}(endpoint.Addr)
		}
	})

	proxy := &redis.ReverseProxy{
		Transport: transport,
		Registry:  endpoints,
		ErrorLog:  log.New(os.Stderr, "proxy hash bench ==> ", 0),
	}

	time.Sleep(time.Second)
	request := redis.NewRequest(endpoints[0].Addr, "SET", redis.List("key", "value"))
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

func makeServerList() (full redis.ServerList, broken redis.ServerList, onedowns []redis.ServerList) {
	full = redis.ServerList{}
	for i := 0; i < 4; i++ {
		// rand.Seed(math.MaxInt32)

		full = append(full, redis.ServerEndpoint{
			Name: "one",
			Addr: fmt.Sprintf("localhost:1%04d", rand.Intn(10000)+i),
		})
	}

	broken = append(full, redis.ServerEndpoint{Name: "zero", Addr: "localhost:0"})

	onedowns = []redis.ServerList{}
	for i := 0; i < len(full); i++ {
		// list containing all but the i'th element of full
		notith := make(redis.ServerList, 0, len(full)-1)
		for j := 0; j < len(full); j++ {
			if j != i {
				notith = append(notith, full[j])
			}
		}
		onedowns = append(onedowns, notith)
	}
	return
}

type responseWriter struct{}

func (w *responseWriter) WriteStream(n int) error {
	return nil
}

func (w *responseWriter) Write(v interface{}) error {
	return nil
}
