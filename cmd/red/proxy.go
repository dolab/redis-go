package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/segmentio/conf"
	consul "github.com/segmentio/consul-go"
	"github.com/segmentio/events"
	eventslog "github.com/segmentio/events/log"
	redis "github.com/segmentio/redis-go"
)

type proxyConfig struct {
	Bind     string `conf:"bind"     help:"Address on which the proxy is listening for incoming connections." validate:"nonzero"`
	Upstream string `conf:"upstream" help:"URL or comma-separated list of upstream servers."                  validate:"nonzero"`
	Debug    bool   `conf:"debug"    help:"Enable debug mode."`
}

func proxy(args []string) (err error) {
	config := proxyConfig{
		Bind: ":6479",
	}

	conf.LoadWith(&config, conf.Loader{
		Name: "red proxy",
		Args: args,
		Sources: []conf.Source{
			conf.NewEnvSource("RED", os.Environ()...),
		},
	})

	events.DefaultLogger.EnableDebug = config.Debug
	events.DefaultLogger.EnableSource = config.Debug

	defer func() {
		if err == nil {
			err = convertPanicToError(recover())
		}
	}()

	lstn := makeListener(config.Bind)
	server := makeProxyServer(config)

	sigchan, sigstop := signals(syscall.SIGINT, syscall.SIGTERM)
	defer sigstop()

	go func() {
		<-sigchan
		sigstop()
		server.Shutdown(context.Background())
	}()

	events.Log("listening on '%{address}s' for incoming connections", lstn.Addr())

	if err = server.Serve(lstn); err == redis.ErrServerClosed {
		err = nil
	}

	return
}

func makeListener(addr string) net.Listener {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		panic(err)
	}
	return l
}

func makeProxyServer(config proxyConfig) *redis.Server {
	logger := eventslog.NewLogger("", 0, events.DefaultHandler)
	return &redis.Server{
		Handler:      makeReverseProxy(config, logger),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  90 * time.Second,
		ErrorLog:     logger,
	}
}

func makeReverseProxy(config proxyConfig, logger *log.Logger) redis.Handler {
	return &redis.ReverseProxy{
		Transport: makeTransport(),
		Registry:  makeRegistry(config.Upstream),
		ErrorLog:  logger,
	}
}

func makeTransport() redis.RoundTripper {
	// TODO: add metrics
	return &redis.Transport{
		ConnsPerHost: 4,
		PingTimeout:  10 * time.Second,
		PingInterval: 15 * time.Second,
	}
}

func makeRegistry(upstream string) (registry redis.ServerRegistry) {
	if strings.Index(upstream, "://") < 0 {
		registry = makeStaticRegistry(upstream)
	} else {
		u, err := url.Parse(upstream)
		if err != nil {
			panic(err)
		}

		switch u.Scheme {
		case "consul":
			registry = makeConsulRegistry(u)

		default:
			panic("unsupported registry: " + u.Scheme)
		}
	}

	return
}

func makeStaticRegistry(upstream string) redis.ServerList {
	addrs := strings.Split(upstream, ",")
	servers := make(redis.ServerList, len(addrs))

	for i, addr := range addrs {
		var name string

		if at := strings.IndexByte(addr, '@'); at < 0 {
			events.Log("adding '%{redis_server_addr}s' to the list of upstream redis servers", addr)
		} else {
			name, addr = addr[:at], addr[at+1:]
			events.Log("adding '%{redis_server_addr}s' as '%{redis_server_name}s' to the list of upstream redis servers", addr, name)
		}

		servers[i] = redis.ServerEndpoint{
			Name: name,
			Addr: addr,
		}
	}

	return servers
}

func makeConsulRegistry(u *url.URL) *consulRegistry {
	v := u.Query()

	r := &consulRegistry{
		service: strings.TrimPrefix(u.Path, "/"),
		cluster: v.Get("cluster"),
		client: &consul.Client{
			Address:    u.Host,
			UserAgent:  fmt.Sprintf("RED (github.com/segmentio/redis-go, version %s)", version),
			Datacenter: v.Get("dc"),
		},
		resolver: &consul.Resolver{
			OnlyPassing: true,
			Cache:       &consul.ResolverCache{},
			Blacklist:   &consul.ResolverBlacklist{},
		},
	}

	if len(r.cluster) != 0 {
		r.resolver.ServiceTags = []string{"redis-cluster:" + r.cluster}
	}

	r.resolver.Client = r.client

	events.Log("using '%{redis_service_name}s' services of the '%{redis_cluster_name}s' from the consul registry at '%{consul_addr}s'",
		r.cluster,
		r.service,
		r.client.Address,
	)

	return r
}

type consulRegistry struct {
	service  string
	cluster  string
	client   *consul.Client
	resolver *consul.Resolver
}

func (r *consulRegistry) LookupServers(ctx context.Context) ([]redis.ServerEndpoint, error) {
	endpoints, err := r.resolver.LookupService(ctx, r.service)
	if err != nil {
		return nil, err
	}

	servers := make([]redis.ServerEndpoint, len(endpoints))

	for i, e := range endpoints {
		servers[i] = redis.ServerEndpoint{
			Name: e.ID,
			Addr: e.Addr.String(),
		}
	}

	return servers, nil
}

func convertPanicToError(v interface{}) error {
	switch x := v.(type) {
	case nil:
		return nil
	case error:
		return x
	case string:
		return errors.New(x)
	default:
		return fmt.Errorf("%v", x)
	}
}
