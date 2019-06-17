package redis

import (
	"context"
)

// The ServerRing interface is an abstraction used to pick a backend redis server
// for key among servers.
type ServerRing interface {
	// LookupServer returns a redis server based on servers and key hashing.
	LookupServer(key string) ServerEndpoint
}

// The ServerRegistry interface is an abstraction used to expose a (potentially
// changing) list of backend redis servers.
type ServerRegistry interface {
	// LookupServers returns a list of redis server endpoints.
	LookupServers(ctx context.Context) (ServerRing, error)
}

// ServerBlacklist is implemented by some ServerRegistry to support black
// listing some server addresses.
type ServerBlacklist interface {
	// Blacklist temporarily blacklists the given server endpoint.
	BlacklistServer(ServerEndpoint)
}

// A ServerRingFunc satisfies the ServerRing interface of custom hashing func.
type ServerRingFunc func(key string) ServerEndpoint

// LookupServer satisfies the ServerRing interface.
func (fn ServerRingFunc) LookupServer(key string) ServerEndpoint {
	return fn(key)
}

// A ServerEndpoint represents a single backend redis server.
type ServerEndpoint struct {
	Name string
	Addr string
}

// LookupServers satisfies the ServerRegistry interface.
func (endpoint ServerEndpoint) LookupServers(ctx context.Context) (ServerRing, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		return ServerRingFunc(func(_ string) ServerEndpoint {
			return endpoint
		}), nil
	}
}

// A ServerList represents a list of backend redis servers.
type ServerList []ServerEndpoint

// LookupServers satisfies the ServerRegistry interface.
func (list ServerList) LookupServers(ctx context.Context) (ServerRing, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		endpoints := make([]ServerEndpoint, len(list))
		copy(endpoints, list)

		return ServerRingFunc(func(key string) ServerEndpoint {
			// TODO: rebuilding the hash ring for every request is not efficient, we should cache and reuse the state.
			ring := NewHashRing(endpoints...)

			return ring.LookupServer(key)
		}), nil
	}
}
