// +build !appengine

package fasthttp

import (
	"bytes"
	"errors"
	"io"
	"mime/multipart"

	"wgo/server"

	"github.com/valyala/fasthttp"
)

type (
	// Request implements `whttp.Request`.
	Request struct {
		*fasthttp.RequestCtx
		header *RequestHeader
		url    *URL
	}
)

// NewRequest returns `Request` instance.
func NewRequest(c *fasthttp.RequestCtx) *Request {
	return &Request{
		RequestCtx: c,
		url:        &URL{URI: c.URI()},
		header:     &RequestHeader{RequestHeader: &c.Request.Header},
	}
}

// IsTLS implements `whttp.Request#TLS` function.
func (r *Request) IsTLS() bool {
	return r.RequestCtx.IsTLS()
}

// Scheme implements `whttp.Request#Scheme` function.
func (r *Request) Scheme() string {
	return string(r.RequestCtx.URI().Scheme())
}

// Host implements `whttp.Request#Host` function.
func (r *Request) Host() string {
	return string(r.RequestCtx.Host())
}

// URL implements `whttp.Request#URL` function.
func (r *Request) URL() server.URL {
	return r.url
}

// Header implements `whttp.Request#Header` function.
func (r *Request) Header() server.Header {
	return r.header
}

// Referer implements `whttp.Request#Referer` function.
func (r *Request) Referer() string {
	return string(r.Request.Header.Referer())
}

// ContentLength implements `whttp.Request#ContentLength` function.
func (r *Request) ContentLength() int64 {
	return int64(r.Request.Header.ContentLength())
}

// UserAgent implements `whttp.Request#UserAgent` function.
func (r *Request) UserAgent() string {
	return string(r.RequestCtx.UserAgent())
}

// RemoteAddress implements `whttp.Request#RemoteAddress` function.
func (r *Request) RemoteAddress() string {
	return r.RemoteAddr().String()
}

// Method implements `whttp.Request#Method` function.
func (r *Request) Method() string {
	return string(r.RequestCtx.Method())
}

// SetMethod implements `whttp.Request#SetMethod` function.
func (r *Request) SetMethod(method string) {
	r.Request.Header.SetMethodBytes([]byte(method))
}

// URI implements `whttp.Request#URI` function.
func (r *Request) URI() string {
	return string(r.RequestURI())
}

// SetURI implements `whttp.Request#SetURI` function.
func (r *Request) SetURI(uri string) {
	r.Request.Header.SetRequestURI(uri)
}

// Body implements `whttp.Request#Body` function.
func (r *Request) Body() io.Reader {
	return bytes.NewBuffer(r.Request.Body())
}

// SetBody implements `whttp.Request#SetBody` function.
func (r *Request) SetBody(reader io.Reader) {
	r.Request.SetBodyStream(reader, 0)
}

// FormValue implements `whttp.Request#FormValue` function.
func (r *Request) FormValue(name string) string {
	return string(r.RequestCtx.FormValue(name))
}

// FormParams implements `whttp.Request#FormParams` function.
func (r *Request) FormParams() (params map[string][]string) {
	params = make(map[string][]string)
	mf, err := r.RequestCtx.MultipartForm()

	if err == fasthttp.ErrNoMultipartForm {
		r.PostArgs().VisitAll(func(k, v []byte) {
			key := string(k)
			if _, ok := params[key]; ok {
				params[key] = append(params[key], string(v))
			} else {
				params[string(k)] = []string{string(v)}
			}
		})
	} else if err == nil {
		for k, v := range mf.Value {
			if len(v) > 0 {
				params[k] = v
			}
		}
	}

	return
}

// FormFile implements `whttp.Request#FormFile` function.
func (r *Request) FormFile(name string) (*multipart.FileHeader, error) {
	return r.RequestCtx.FormFile(name)
}

// MultipartForm implements `whttp.Request#MultipartForm` function.
func (r *Request) MultipartForm() (*multipart.Form, error) {
	return r.RequestCtx.MultipartForm()
}

// Cookie implements `whttp.Request#Cookie` function.
func (r *Request) Cookie(name string) (server.Cookie, error) {
	c := new(fasthttp.Cookie)
	b := r.Request.Header.Cookie(name)
	if b == nil {
		//return nil, wgo.ErrCookieNotFound
		return nil, errors.New("err cookie not found")
	}
	c.SetKey(name)
	c.SetValueBytes(b)
	return &Cookie{c}, nil
}

// Cookies implements `whttp.Request#Cookies` function.
func (r *Request) Cookies() []server.Cookie {
	cookies := []server.Cookie{}
	r.Request.Header.VisitAllCookie(func(name, value []byte) {
		c := new(fasthttp.Cookie)
		c.SetKeyBytes(name)
		c.SetValueBytes(value)
		cookies = append(cookies, &Cookie{c})
	})
	return cookies
}

func (r *Request) reset(c *fasthttp.RequestCtx, h *RequestHeader, u *URL) {
	r.RequestCtx = c
	r.header = h
	r.url = u
}
