package redis

// The ServerCommander interface is an abstraction used to extend redis command.
type ServerCommander interface {
	LookupCommanders() map[string]Commander
}

// A Commander responds to a Redis CMD.
//
// Except for reading the argument list, handlers should not modify the provided
// Request.
type Commander interface {
	ServeCommand(tripper RoundTripper, hashing ServerRing, w ResponseWriter, r *Request)
}

// The CommanderFunc type is an adapter to allow the use of ordinary functions as
// redis commanders. If fn is a function with the appropriate signature.
type CommanderFunc func(tripper RoundTripper, hashing ServerRing, w ResponseWriter, r *Request)

func (fn CommanderFunc) ServeCommand(tripper RoundTripper, hashing ServerRing, w ResponseWriter, r *Request) {
	fn(tripper, hashing, w, r)
}
