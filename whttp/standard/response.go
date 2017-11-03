package standard

import (
	"bufio"
	"bytes"
	"io"
	"net"
	"net/http"

	"wgo/server"
	"wgo/utils"
)

type (
	// Response implements `whttp.Response`.
	Response struct {
		http.ResponseWriter
		adapter   *responseAdapter
		header    *Header
		status    int
		size      int64
		committed bool
		writer    io.Writer
		buffer    *bytes.Buffer
	}

	responseAdapter struct {
		*Response
	}
)

// NewResponse returns `Response` instance.
//func NewResponse(w http.ResponseWriter) (r *Response) {
func NewResponse() (r *Response) {
	r = &Response{
		//ResponseWriter: w,
		//writer:         w,
		header: &Header{Header: make(http.Header)},
		buffer: bytes.NewBuffer([]byte{}),
	}
	r.adapter = &responseAdapter{Response: r}
	return
}

// CopyTo copies resp contents to dst except of body stream.
func (r *Response) CopyTo(dst interface{}) {
	// copy header
	//rw := r.ResponseWriter
	//dst.(*Response).reset(rw, r.adapter, r.header)
	copyHeader(dst.(*Response).header, r.header)
	//r.header.CopyTo(dst.header.ResponseHeader)
	// copy body
	//r.RequestCtx.Response.CopyTo(&dst.RequestCtx.Response)
	*dst.(*Response).buffer = *r.buffer
}
func copyHeader(dst, src *Header) {
	for k, vv := range src.Header {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

// Header implements `whttp.Response#Header` function.
func (r *Response) Header() server.Header {
	return r.header
}

// WriteHeader implements `whttp.Response#WriteHeader` function.
func (r *Response) WriteHeader(code int) {
	if r.committed {
		//r.logger.Warn("response already committed")
		return
	}
	r.status = code
	//r.ResponseWriter.WriteHeader(code)   //注释掉, 这里只设置状态
	r.committed = true
}

// Write implements `whttp.Response#Write` function.
func (r *Response) Write(b []byte) (n int, err error) {
	//if !r.committed {
	//	r.WriteHeader(http.StatusOK)
	//}
	n, err = r.writer.Write(b)
	//r.size += int64(n)
	return
}

func (r *Response) WriteGzip(b []byte) (n int, err error) {
	//if !r.committed {
	//	r.WriteHeader(http.StatusOK)
	//}
	n, err = utils.WriteGzip(r.writer, b)
	//r.size += int64(n)
	return
}

// SetCookie implements `whttp.Response#SetCookie` function.
func (r *Response) SetCookie(c server.Cookie) {
	http.SetCookie(r.ResponseWriter, &http.Cookie{
		Name:     c.Name(),
		Value:    c.Value(),
		Path:     c.Path(),
		Domain:   c.Domain(),
		Expires:  c.Expires(),
		Secure:   c.Secure(),
		HttpOnly: c.HTTPOnly(),
	})
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

// Flush implements the http.Flusher interface to allow an HTTP handler to flush
// buffered data to the client.
// See https://golang.org/pkg/net/http/#Flusher
func (r *Response) Flush() {
	r.ResponseWriter.WriteHeader(r.status)
	//r.ResponseWriter.Write(r.buffer.Bytes())
	n, _ := r.buffer.WriteTo(r.ResponseWriter)
	r.size += n
	//r.ResponseWriter.(http.Flusher).Flush()	// 这行代码会导致没有Content-Length, 被`Transfer-Encoding: chunked`取代
}

// Hijack implements the http.Hijacker interface to allow an HTTP handler to
// take over the connection.
// See https://golang.org/pkg/net/http/#Hijacker
func (r *Response) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return r.ResponseWriter.(http.Hijacker).Hijack()
}

// CloseNotify implements the http.CloseNotifier interface to allow detecting
// when the underlying connection has gone away.
// This mechanism can be used to cancel long operations on the server if the
// client has disconnected before the response is ready.
// See https://golang.org/pkg/net/http/#CloseNotifier
func (r *Response) CloseNotify() <-chan bool {
	return r.ResponseWriter.(http.CloseNotifier).CloseNotify()
}

func (r *Response) reset(w http.ResponseWriter, a *responseAdapter, h *Header) {
	r.ResponseWriter = w
	r.adapter = a
	r.header = h
	r.status = http.StatusOK
	r.size = 0
	r.committed = false
	//r.writer = w
	r.buffer.Reset()
	r.writer = r.buffer
}

func (r *responseAdapter) Header() http.Header {
	return r.ResponseWriter.Header()
}

func (r *responseAdapter) reset(res *Response) {
	r.Response = res
}
