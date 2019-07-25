package redistest

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/dolab/redis-go"
	"github.com/dolab/redis-go/metrics"

	"k8s.io/apimachinery/pkg/util/wait"
)

func TestServer(serverList redis.ServerList, handlers ...func(w redis.ResponseWriter, r *redis.Request)) <-chan struct{} {
	allServers := map[string]bool{}
	for _, endpoint := range serverList {
		allServers[endpoint.Addr] = true

		var handler redis.HandlerFunc
		if len(handlers) > 0 {
			handler = handlers[0]
		} else {
			handler = TestServerHandler()
		}

		go func(addr string, handler redis.Handler) {
			server := &redis.Server{
				Addr:         addr,
				Handler:      handler,
				ReadTimeout:  1 * time.Minute,
				WriteTimeout: 1 * time.Minute,
				IdleTimeout:  5 * time.Minute,
				ErrorLog:     log.New(os.Stdout, "[Test Backend Server] ", os.O_CREATE|os.O_WRONLY|os.O_APPEND),
			}
			log.Fatal(server.ListenAndServe())
		}(endpoint.Addr, handler)
	}

	stopCh := make(chan struct{})
	wait.Until(func() {
		if len(allServers) == 0 {
			close(stopCh)
		}

		for addr := range allServers {
			client := redis.Client{
				Addr:    addr,
				Timeout: 10 * time.Millisecond,
			}

			err := client.Exec(context.Background(), "PING")
			if err == nil {
				delete(allServers, addr)
			}
		}
	}, 10*time.Millisecond, stopCh)

	return stopCh
}

func TestServerHandler() redis.HandlerFunc {
	localStore := sync.Map{}

	return func(w redis.ResponseWriter, r *redis.Request) {
		for _, cmd := range r.Cmds {
			switch cmd.Cmd {
			case "PING":
				w.Write("OK")

			case "SET":
				var (
					dst string

					args []string
				)
				for cmd.Args.Next(&dst) {
					args = append(args, dst)
				}

				if len(args) > 0 {
					if len(args) > 1 {
						localStore.Store(args[0], args[1:])
					} else {
						localStore.Store(args[0], nil)
					}
				}

				w.Write("")

			case "GET":
				w.WriteStream(cmd.Args.Len())

				var (
					dst string
				)
				for cmd.Args.Next(&dst) {
					v, ok := localStore.Load(dst)
					if !ok {
						w.Write("")
					} else {
						vals, ok := v.([]string)
						if ok {
							w.Write(strings.Join(vals, " "))
						} else {
							w.Write(fmt.Sprintf("%v", v))
						}
					}
				}
			}
		}
	}
}

func FakeServer(handler redis.Handler) (srv *redis.Server, url string) {
	return FakeTimeoutServer(handler, 1000*time.Millisecond)
}

func FakeTimeoutServer(handler redis.Handler, timeout time.Duration) (srv *redis.Server, addr string) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}

	srv = &redis.Server{
		Handler:      handler,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  timeout,
		ErrorLog:     log.New(os.Stdout, "[Server Timeout] ", os.O_CREATE|os.O_WRONLY|os.O_APPEND),
	}

	go func() {
		err := srv.Serve(l)
		if err != redis.ErrServerClosed {
			log.Fatalf("[Server] %v", err)
		}
	}()

	addr = l.Addr().String()

	stopCh := make(chan struct{})
	wait.Until(func() {
		client := redis.Client{
			Addr:    addr,
			Timeout: 10 * time.Millisecond,
		}

		err := client.Exec(context.Background(), "PING")
		if err == nil {
			close(stopCh)
		}
	}, 10*time.Millisecond, stopCh)
	<-stopCh

	return
}

func FakeMetricsServer(handler redis.Handler, timeout time.Duration) (srv *redis.Server, addr string) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}

	srv = &redis.Server{
		Handler:      handler,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  timeout,
		ErrorLog:     log.New(os.Stdout, "[Server Metrics] ", os.O_CREATE|os.O_WRONLY|os.O_APPEND),
	}
	srv.WithMetrics(metrics.Options{
		Subsystem:           "proxy",
		Labels:              nil,
		EnableServerMetrics: true,
	})

	go func() {
		err := srv.Serve(l)
		if err != redis.ErrServerClosed {
			log.Fatalf("[Server] %v", err)
		}
	}()

	addr = l.Addr().String()

	stopCh := make(chan struct{})
	wait.Until(func() {
		client := redis.Client{
			Addr:    addr,
			Timeout: 10 * time.Millisecond,
		}

		err := client.Exec(context.Background(), "PING")
		if err == nil {
			close(stopCh)
		}
	}, 10*time.Millisecond, stopCh)
	<-stopCh

	return
}

func FakeServerList() (validServers redis.ServerList, brokenServers redis.ServerList, oneDownServers []redis.ServerList) {
	validServers = redis.ServerList{}
	for i := 0; i < 4; i++ {
		validServers = append(validServers, redis.ServerEndpoint{
			Name: fmt.Sprintf("server-%d", i),
			Addr: fmt.Sprintf("127.0.0.1:1%04d", rand.Intn(10000)+i),
		})
	}

	brokenServers = append(redis.ServerList{}, redis.ServerEndpoint{Name: "zero", Addr: "localhost:0"})

	oneDownServers = []redis.ServerList{}
	for i := 0; i < len(validServers); i++ {
		// list containing all but the i'th element of validServers
		notith := make(redis.ServerList, 0, len(validServers)-1)
		for j := 0; j < len(validServers); j++ {
			if j != i {
				notith = append(notith, validServers[j])
			}
		}

		oneDownServers = append(oneDownServers, notith)
	}
	return
}
