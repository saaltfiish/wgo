package whttp

import (
	"io"
	"mime/multipart"
	"time"

	"wgo/environ"
	"wgo/server"

	ctx "golang.org/x/net/context"
)

type (
	// http context
	Context interface {
		// Context returns `net/ctx.Context`.
		Context() ctx.Context

		// SetContext sets `net/ctx.Context`.
		SetContext(ctx.Context)

		// Deadline returns the time when work done on behalf of this context
		// should be canceled.  Deadline returns ok==false when no deadline is
		// set.  Successive calls to Deadline return the same results.
		Deadline() (deadline time.Time, ok bool)

		// Done returns a channel that's closed when work done on behalf of this
		// context should be canceled.  Done may return nil if this context can
		// never be canceled.  Successive calls to Done return the same value.
		Done() <-chan struct{}

		// Err returns a non-nil error value after Done is closed.  Err returns
		// Canceled if the context was canceled or DeadlineExceeded if the
		// context's deadline passed.  No other values for Err are defined.
		// After Done is closed, successive calls to Err return the same value.
		Err() error

		// Value returns the value associated with this context for key, or nil
		// if no value is associated with key.  Successive calls to Value with
		// the same key returns the same result.
		Value(key interface{}) interface{}

		// Request returns `Request` interface.
		Request() interface{}

		// Request returns `Response` interface.
		Response() interface{}

		// Path returns the registered path for the handler.
		Path() string

		// SetPath sets the registered path for the handler.
		SetPath(string)

		// P returns path parameter by index.
		P(int) string

		// Param returns path parameter by name.
		Param(string) string

		// ParamNames returns path parameter names.
		ParamNames() []string

		// SetParamNames sets path parameter names.
		SetParamNames(...string)

		// ParamValues returns path parameter values.
		ParamValues() []string

		// SetParamValues sets path parameter values.
		SetParamValues(...string)

		// QueryParam returns the query param for the provided name. It is an alias
		// for `URL#QueryParam()`.
		QueryParam(string) string

		// QueryParams returns the query parameters as map.
		// It is an alias for `URL#QueryParams()`.
		QueryParams() map[string][]string

		// FormValue returns the form field value for the provided name. It is an
		// alias for `Request#FormValue()`.
		FormValue(string) string

		// FormParams returns the form parameters as map.
		// It is an alias for `Request#FormParams()`.
		FormParams() map[string][]string

		// FormFile returns the multipart form file for the provided name. It is an
		// alias for `Request#FormFile()`.
		FormFile(string) (*multipart.FileHeader, error)

		// MultipartForm returns the multipart form.
		// It is an alias for `Request#MultipartForm()`.
		MultipartForm() (*multipart.Form, error)

		// Cookie returns the named cookie provided in the request.
		// It is an alias for `Request#Cookie()`.
		Cookie(string) (server.Cookie, error)

		// SetCookie adds a `Set-Cookie` header in HTTP response.
		// It is an alias for `Response#SetCookie()`.
		SetCookie(server.Cookie)

		// Cookies returns the HTTP cookies sent with the request.
		// It is an alias for `Request#Cookies()`.
		Cookies() []server.Cookie

		// Get retrieves data from the context.
		Get(string) interface{}

		// Set saves data in the context.
		Set(string, interface{})

		// Bind binds the request body into provided type `i`. The default binder
		// does it based on Content-Type header.
		Bind(interface{}) error

		// return error
		ERROR(error)
		// HTML sends an HTTP response with status code.
		HTML(code int, html string) error

		// HTMLBlob sends an HTTP blob response with status code.
		HTMLBlob(code int, b []byte) error

		// String sends a string response with status code.
		String(code int, s string) error

		// JSON sends a JSON response with status code.
		JSON(code int, i interface{}) error

		// JSONPretty sends a pretty-print JSON with status code.
		JSONPretty(code int, i interface{}, indent string) error

		// JSONBlob sends a JSON blob response with status code.
		JSONBlob(code int, b []byte) error

		// JSONP sends a JSONP response with status code. It uses `callback` to construct
		// the JSONP payload.
		JSONP(code int, callback string, i interface{}) error

		// JSONPBlob sends a JSONP blob response with status code. It uses `callback`
		// to construct the JSONP payload.
		JSONPBlob(code int, callback string, b []byte) error

		// XML sends an XML response with status code.
		XML(code int, i interface{}) error

		// XMLPretty sends a pretty-print XML with status code.
		XMLPretty(code int, i interface{}, indent string) error

		// XMLBlob sends an XML blob response with status code.
		XMLBlob(code int, b []byte) error

		// Blob sends a blob response with status code and content type.
		Blob(code int, contentType string, b []byte) error

		// Stream sends a streaming response with status code and content type.
		Stream(code int, contentType string, r io.Reader) error

		// File sends a response with the content of the file.
		File(string) error

		// Attachment sends a response from `io.ReaderSeeker` as attachment, prompting
		// client to save the file.
		Attachment(io.ReadSeeker, string) error

		// NoContent sends a response with no body and a status code.
		NoContent(int) error

		// Redirect redirects the request with status code.
		Redirect(int, string) error

		// Logger returns the `Logger` instance.
		SetLogger(interface{})
		Logger() server.Logger

		// ServeContent sends static content from `io.Reader` and handles caching
		// via `If-Modified-Since` request header. It automatically sets `Content-Type`
		// and `Last-Modified` response headers.
		ServeContent(io.ReadSeeker, string, time.Time) error

		// Start return start time of get context
		Start() time.Time
		// Sub return duration from start
		Sub() time.Duration
		// Ext return ext content
		SetExt(interface{})
		Ext() interface{}
		// request id
		SetRequestID(string)
		RequestID() string

		// HTTPReset resets the context after request completes. It must be called along
		// See `Mux#Serve()`
		HTTPReset(Request, Response)

		// mux
		SetMux(server.Mux)
		Mux() server.Mux

		// useful mehtods
		ServerMode() string
		Cfg() *environ.Config
		Host() string
		Depth() uint64
		ClientIP() string

		//logging
		Debug(arg0 interface{}, args ...interface{})
		Info(arg0 interface{}, args ...interface{})
		Warn(arg0 interface{}, args ...interface{})
		Error(arg0 interface{}, args ...interface{})

		// router
		SetNode(interface{})
		Node() interface{}

		// encoding
		Encoding() string
	}

	// 生成context
	ContextGenFuc func(Request, Response) Context
)
