package redis

import (
	"sort"

	"github.com/segmentio/fasthash/jody"
)

const (
	maxRingReplication = 40
)

const (
	serverListRingHashing ringHashingKey = iota
)

type ringHashingKey int

type ringNode struct {
	endpoint ServerEndpoint
	hash     uint64
}

// hashRing is the implementation of a consistent hashing distribution of string
// keys to server addresses.
type hashRing []ringNode

func NewHashRing(endpoints ...ServerEndpoint) ServerRing {
	if len(endpoints) == 0 {
		return nil
	}

	ring := make(hashRing, 0, maxRingReplication*len(endpoints))

	for _, endpoint := range endpoints {
		h := jody.HashString64(endpoint.Addr)

		for i := 0; i != maxRingReplication; i++ {
			ring = append(ring, ringNode{
				endpoint: endpoint,
				hash:     consistentHash(jody.AddUint64(h, uint64(i))),
			})
		}
	}

	sort.Sort(ring)

	return ring
}

func (r hashRing) LookupServer(key string) ServerEndpoint {
	n := len(r)
	h := consistentHash(jody.HashString64(key))
	i := sort.Search(n, func(i int) bool { return h < r[i].hash })

	if i == n {
		i = 0
	}

	return r[i].endpoint
}

func (r hashRing) Len() int {
	return len(r)
}

func (r hashRing) Less(i int, j int) bool {
	return r[i].hash < r[j].hash
}

func (r hashRing) Swap(i int, j int) {
	r[i], r[j] = r[j], r[i]
}

func consistentHash(h uint64) uint64 {
	const radix = 1e9
	return h % radix
}
