package redistest

import (
	"context"
	"testing"
	"time"

	"github.com/golib/assert"

	"github.com/dolab/redis-go"
)

// MakeServerRegistry is the type of factory functions that the
// TestServerRegistry test suite uses to create Clients to run the
// tests against.
type MakeServerRegistry func() (redis.ServerRegistry, string, redis.ServerEndpoint, func(), error)

// TestServerRegistry is a test suite which verifies the behavior of
// ServerRegistry implementations.
func TestServerRegistry(t *testing.T, makeServerRegistry MakeServerRegistry) {
	tests := []struct {
		scenario string
		function func(*testing.T, context.Context, redis.ServerRegistry, string, redis.ServerEndpoint)
	}{
		{
			scenario: "calling LookupServers with a canceled context returns an error",
			function: testServerRegistryCancel,
		},
		{
			scenario: "calling LookupServers returns the expected list of servers",
			function: testServerRegistryLookupServers,
		},
	}

	for _, test := range tests {
		testFunc := test.function

		t.Run(test.scenario, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			registry, key, endpoint, callback, err := makeServerRegistry()
			if err != nil {
				t.Fatal(err)
			}
			defer callback()

			testFunc(t, ctx, registry, key, endpoint)
		})
	}
}

func testServerRegistryCancel(t *testing.T, ctx context.Context, registry redis.ServerRegistry, key string, endpoints redis.ServerEndpoint) {
	it := assert.New(t)

	ctx, cancel := context.WithCancel(ctx)
	cancel()

	it.Nil(registry.LookupServers(ctx))
}

func testServerRegistryLookupServers(t *testing.T, ctx context.Context, registry redis.ServerRegistry, key string, endpoint redis.ServerEndpoint) {
	it := assert.New(t)

	ring, err := registry.LookupServers(ctx)
	if it.Nil(err) {
		it.Implements((*redis.ServerRing)(nil), ring)
		it.Equal(endpoint, ring.LookupServer(key))
	}
}
