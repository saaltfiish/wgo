package wrpc

import (
	"context"
	"reflect"
	"runtime"

	"google.golang.org/grpc"
	// "google.golang.org/grpc/transport"
)

type (
	Request struct {
		Body interface{}

		method  string
		header  *Header
		context context.Context
	}

	Response struct {
		Body interface{}
		Err  error

		header *Header
	}

	// HandlerFunc is an adapter to allow the use of `func(Context)` as an HTTP handler.
	// HandlerFunc defines a function to server HTTP requests.
	HandlerFunc func(Context) error

	// MiddlewareFunc defines a function to process middleware.
	MiddlewareFunc func(HandlerFunc) HandlerFunc

	Middleware struct {
		tag  string // 标识
		Func MiddlewareFunc
	}
)

func (h MiddlewareFunc) Name() string {
	t := reflect.ValueOf(h).Type()
	if t.Kind() == reflect.Func {
		return runtime.FuncForPC(reflect.ValueOf(h).Pointer()).Name()
	}
	return t.String()
}

// new middleware
func NewMiddleware(tag string, m MiddlewareFunc) *Middleware {
	return &Middleware{
		tag:  tag,
		Func: m,
	}
}

// get request context
func (req *Request) Context() context.Context {
	return req.context
}

// get method name
func (req *Request) Method() string {
	return req.method
}

// get query
func (req *Request) Query() string {
	//return req.method
	// if stream, ok := transport.StreamFromContext(req.context); ok {
	// 	return stream.Method()
	// }
	return ""
}

// get request headers
func (req *Request) Header() *Header {
	return req.header
}

// get request ip
func (req *Request) RemoteAddress() string {
	// if stream, ok := transport.StreamFromContext(req.context); ok {
	// 	if t := stream.ServerTransport(); t != nil {
	// 		return t.RemoteAddr().String()
	// 	}
	// }
	return ""
}

// get response headers
func (res *Response) Header() *Header {
	return res.header
}

// flush
func (res *Response) Flush(c context.Context) {
	grpc.SetHeader(c, res.header.MD)
}
