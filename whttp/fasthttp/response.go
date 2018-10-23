// +build !appengine

package fasthttp

import (
	"fmt"
	"io"
	"net/http"

	"wgo/server"
	"wgo/utils"

	"github.com/valyala/fasthttp"
)

type (
	// Response implements `whttp.Response`.
	Response struct {
		*fasthttp.RequestCtx
		header    *ResponseHeader
		status    int
		size      int64
		committed bool
		writer    io.Writer
	}
)

// NewResponse returns `Response` instance.
func NewResponse() *Response {
	c := new(fasthttp.RequestCtx)
	return &Response{
		RequestCtx: c,
		header:     &ResponseHeader{ResponseHeader: &c.Response.Header},
		writer:     c,
	}
}

// CopyTo copies resp contents to dst except of body stream.
func (r *Response) CopyTo(dst interface{}) {
	// copy header
	r.header.CopyTo(dst.(*Response).header.ResponseHeader)
	// copy body
	r.RequestCtx.Response.CopyTo(&dst.(*Response).RequestCtx.Response)
}

// Header implements `server.Response#Header` function.
func (r *Response) Header() server.Header {
	return r.header
}

// WriteHeader implements `whttp.Response#WriteHeader` function.
func (r *Response) WriteHeader(code int) {
	if r.committed {
		return
	}
	r.status = code
	r.SetStatusCode(code)
	r.committed = true
}

// Write implements `whttp.Response#Write` function.
func (r *Response) Write(b []byte) (n int, err error) {
	if !r.committed {
		r.WriteHeader(http.StatusOK)
	}
	n, err = r.writer.Write(b)
	r.size += int64(n)
	return
}

func (r *Response) WriteGzip(b []byte) (n int, err error) {
	if !r.committed {
		r.WriteHeader(http.StatusOK)
	}
	n, err = utils.WriteGzip(r.writer, b)
	r.size += int64(n)
	return
}

// SetCookie implements `whttp.Response#SetCookie` function.
func (r *Response) SetCookie(c server.Cookie) {
	cookie := new(fasthttp.Cookie)
	cookie.SetKey(c.Name())
	cookie.SetValue(c.Value())
	cookie.SetPath(c.Path())
	cookie.SetDomain(c.Domain())
	cookie.SetExpire(c.Expires())
	cookie.SetSecure(c.Secure())
	cookie.SetHTTPOnly(c.HTTPOnly())
	r.Response.Header.SetCookie(cookie)
}

// Status implements `whttp.Response#Status` function.
func (r *Response) Status() int {
	return r.status
}

// Size implements `whttp.Response#Size` function.
func (r *Response) Size() int64 {
	return r.size
}

// Committed implements `whttp.Response#Committed` function.
func (r *Response) Committed() bool {
	return r.committed
}

// Writer implements `whttp.Response#Writer` function.
func (r *Response) Writer() io.Writer {
	return r.writer
}

// SetWriter implements `whttp.Response#SetWriter` function.
func (r *Response) SetWriter(w io.Writer) {
	r.writer = w
}

// flush count size, for gzip
func (r *Response) Flush() {
	r.size = int64(len(r.Body()))
	fmt.Printf("[fasthttp.Flush]size: %d\n", r.size)
}

// Body implements `whttp.Response#Body` function.
func (r *Response) Body() []byte {
	return r.RequestCtx.Response.Body()
}

func (r *Response) reset(c *fasthttp.RequestCtx, h *ResponseHeader) {
	r.RequestCtx = c
	r.header = h
	r.status = http.StatusOK
	r.size = 0
	r.committed = false
	r.writer = c
}
