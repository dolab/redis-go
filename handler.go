package redis

// The ServerHandler interface is an abstraction used to extend redis handler.
type ServerHandler interface {
	LookupHandlers() map[string]Handler
}

// A Handler responds to a Redis request.
//
// ServeRedis should write reply headers and data to the ResponseWriter and then
// return. Returning signals that the request is finished; it is not valid to
// use the ResponseWriter or read from the Request.Args after or concurrently with
// the completion of the ServeRedis call.
//
// Except for reading the argument list, handlers should not modify the provided
// Request.
type Handler interface {
	// ServeRedis is called by a Redis server to handle requests.
	ServeRedis(ResponseWriter, *Request)
}

// The HandlerFunc type is an adapter to allow the use of ordinary functions as
// Redis handlers. If f is a function with the appropriate signature.
type HandlerFunc func(ResponseWriter, *Request)

// ServeRedis implements the Handler interface, calling f.
func (fn HandlerFunc) ServeRedis(res ResponseWriter, req *Request) {
	fn(res, req)
}
