package redis

import (
	"strings"
)

// Response represents the response from a Redis request.
type Response struct {
	// Args is the arguments list of the response.
	//
	// When the response is obtained from Transport.RoundTrip or from Client.Do
	// the Args field is never nil.
	Args Args

	// TxArgs is the argument list of response to requests that were sent as
	// transactions.
	TxArgs TxArgs

	// request is the request that was sent to obtain this Response. It used by retry
	// to return a retryable request.
	request *Request

	// whether redis response with an error message, RESP defines prefix with `-`.
	respErr bool
}

// IsRespError returns true if redis response with an error message. You can get error by calling
// response.Close() or build a new request by calling response.Retry() for retrying.
func (resp *Response) IsRespError() bool {
	return resp.respErr
}

// retry returns a new *Request if the response is retryable and nil. Otherwise, it returns an error
// indicates the request CANNOT apply retry.
func (resp *Response) Retry() (req *Request, err error) {
	if !resp.respErr {
		err = ErrNotRetryable
		return
	}

	err = resp.Close()
	if err == nil {
		err = ErrNotRetryable
		return
	}

	s := err.Error()

	if len(s) <= 5 || s[:5] != "MOVED" {
		err = ErrNotRetryable
		return
	}

	tmp := strings.SplitN(s, " ", 3)
	if len(tmp) != 3 {
		err = ErrNotRetryable
		return
	}

	// fill with new data for retrying
	req, err = resp.request.newRequest()
	if err != nil {
		return
	}
	req.Addr = tmp[2]

	return
}

// Close closes all arguments of the response.
func (resp *Response) Close() error {
	var err error

	if resp.Args != nil {
		err = resp.Args.Close()
	}

	if resp.TxArgs != nil {
		err = resp.TxArgs.Close()
	}

	return err
}
