package whttp

import (
	"io"
	"mime/multipart"
	"reflect"
	"runtime"

	"wgo/server"
	"wgo/whttp/fasthttp"
	"wgo/whttp/standard"
)

type (
	// Request defines the interface for HTTP request.
	Request interface {
		// IsTLS returns true if HTTP connection is TLS otherwise false.
		IsTLS() bool

		// Scheme returns the HTTP protocol scheme, `http` or `https`.
		Scheme() string

		// Host returns HTTP request host. Per RFC 2616, this is either the value of
		// the `Host` header or the host name given in the URL itself.
		Host() string

		// URI returns the unmodified `Request-URI` sent by the client.
		URI() string

		// SetURI sets the URI of the request.
		SetURI(string)

		// URL returns `engine.URL`.
		URL() server.URL

		// Header returns `engine.Header`.
		Header() server.Header

		// Referer returns the referring URL, if sent in the request.
		Referer() string

		// Protocol returns the protocol version string of the HTTP request.
		// Protocol() string

		// ProtocolMajor returns the major protocol version of the HTTP request.
		// ProtocolMajor() int

		// ProtocolMinor returns the minor protocol version of the HTTP request.
		// ProtocolMinor() int

		// ContentLength returns the size of request's body.
		ContentLength() int64

		// UserAgent returns the client's `User-Agent`.
		UserAgent() string

		// RemoteAddress returns the client's network address.
		RemoteAddress() string

		// Method returns the request's HTTP function.
		Method() string

		// SetMethod sets the HTTP method of the request.
		SetMethod(string)

		// Body returns request's body.
		Body() io.Reader

		// Body sets request's body.
		SetBody(io.Reader)

		// FormValue returns the form field value for the provided name.
		FormValue(string) string

		// FormParams returns the form parameters.
		FormParams() map[string][]string

		// FormFile returns the multipart form file for the provided name.
		FormFile(string) (*multipart.FileHeader, error)

		// MultipartForm returns the multipart form.
		MultipartForm() (*multipart.Form, error)

		// Cookie returns the named cookie provided in the request.
		Cookie(string) (server.Cookie, error)

		// Cookies returns the HTTP cookies sent with the request.
		Cookies() []server.Cookie
	}

	// Response defines the interface for HTTP response.
	Response interface {
		// Header returns `engine.Header`
		Header() server.Header

		// WriteHeader sends an HTTP response header with status code.
		WriteHeader(int)

		// Write writes the data to the connection as part of an HTTP reply.
		Write(b []byte) (int, error)
		// WriteGzip writes the gziped data to the connection as part of an HTTP reply.
		WriteGzip(b []byte) (int, error)

		// SetCookie adds a `Set-Cookie` header in HTTP response.
		SetCookie(server.Cookie)

		// Status returns the HTTP response status.
		Status() int

		// Size returns the number of bytes written to HTTP response.
		Size() int64

		// Committed returns true if HTTP response header is written, otherwise false.
		Committed() bool
		Commit()

		// Flush
		Flush()

		// Write returns the HTTP response writer.
		Writer() io.Writer

		// SetWriter sets the HTTP response writer.
		SetWriter(io.Writer)

		// response copy
		CopyTo(interface{})
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

// newEngine
func newEngine(name string) server.Engine {
	// Debug("[whttp.newEngine]name: %s", name)
	switch name {
	case "standard":
		return standard.New()
	default:
		return fasthttp.New()
	}
}

// engine vendor
// 把能生成for whttp的 server.Engine 放到server.Server
func Factory(s *server.Server, cgen func() interface{}, mconv func(...interface{}) []*Middleware) *server.Server {
	// Debug("[whttp.Factory]engine: %s", s.EngineName())
	// engine factory func
	var ef server.EngineFactory
	ef = func() server.Engine {
		return newEngine(s.EngineName())
	}
	// mux factory func
	var mf server.MuxFactory
	mf = func() server.Mux {
		return NewMux(s.EngineName(), cgen, mconv)
	}
	return s.Factory(ef, mf)
}
