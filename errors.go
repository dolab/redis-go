package redis

import "errors"

var (
	// ErrServerClosed is returned by Server.Serve when the server is closed.
	ErrNil                           = errors.New("redis: nil")
	ErrNilArgs                       = errors.New("cannot parse values from a nil argument list")
	ErrServerClosed                  = errors.New("redis: Server closed")
	ErrNegativeStreamCount           = errors.New("invalid call to redis.ResponseWriter.WriteStream with a negative value")
	ErrWriteStreamCalledAfterWrite   = errors.New("invalid call to redis.ResponseWriter.WriteStream after redis.ResponseWriter.Write was called")
	ErrWriteStreamCalledTooManyTimes = errors.New("multiple calls to ResponseWriter.WriteStream")
	ErrWriteCalledTooManyTimes       = errors.New("too many calls to redis.ResponseWriter.Write")
	ErrWriteCalledNotEnoughTimes     = errors.New("not enough calls to redis.ResponseWriter.Write")
	ErrHijacked                      = errors.New("invalid use of a hijacked redis.ResponseWriter")
	ErrNotHijackable                 = errors.New("the response writer is not hijackable")
	ErrNotRetryable                  = errors.New("the request cannot retry")
	ErrNotPipeline                   = errors.New("redis: not pipeline")
)
