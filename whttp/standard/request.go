package standard

import (
	"errors"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"strings"

	"wgo/server"
)

type (
	// Request implements `whttp.Request`.
	Request struct {
		*http.Request
		header *Header
		url    *URL
	}
)

const (
	defaultMemory = 32 << 20 // 32 MB
)

// NewRequest returns `Request` instance.
func NewRequest(r *http.Request) *Request {
	return &Request{
		Request: r,
		url:     &URL{URL: r.URL},
		header:  &Header{Header: r.Header},
	}
}

// IsTLS implements `whttp.Request#TLS` function.
func (r *Request) IsTLS() bool {
	return r.Request.TLS != nil
}

// Scheme implements `whttp.Request#Scheme` function.
func (r *Request) Scheme() string {
	// Can't use `r.Request.URL.Scheme`
	// See: https://groups.google.com/forum/#!topic/golang-nuts/pMUkBlQBDF0
	if r.IsTLS() {
		return "https"
	}
	return "http"
}

// Host implements `whttp.Request#Host` function.
func (r *Request) Host() string {
	return r.Request.Host
}

// URL implements `whttp.Request#URL` function.
func (r *Request) URL() server.URL {
	return r.url
}

// Header implements `whttp.Request#URL` function.
func (r *Request) Header() server.Header {
	return r.header
}

// Referer implements `whttp.Request#Referer` function.
func (r *Request) Referer() string {
	return r.Request.Referer()
}

// func Proto() string {
// 	return r.request.Proto()
// }
//
// func ProtoMajor() int {
// 	return r.request.ProtoMajor()
// }
//
// func ProtoMinor() int {
// 	return r.request.ProtoMinor()
// }

// ContentLength implements `whttp.Request#ContentLength` function.
func (r *Request) ContentLength() int64 {
	return r.Request.ContentLength
}

// UserAgent implements `whttp.Request#UserAgent` function.
func (r *Request) UserAgent() string {
	return r.Request.UserAgent()
}

// RemoteAddress implements `whttp.Request#RemoteAddress` function.
func (r *Request) RemoteAddress() string {
	return r.RemoteAddr
}

// Method implements `whttp.Request#Method` function.
func (r *Request) Method() string {
	return r.Request.Method
}

// SetMethod implements `whttp.Request#SetMethod` function.
func (r *Request) SetMethod(method string) {
	r.Request.Method = method
}

// URI implements `whttp.Request#URI` function.
func (r *Request) URI() string {
	return r.RequestURI
}

// SetURI implements `whttp.Request#SetURI` function.
func (r *Request) SetURI(uri string) {
	r.RequestURI = uri
}

// Body implements `whttp.Request#Body` function.
func (r *Request) Body() io.Reader {
	return r.Request.Body
}

// SetBody implements `whttp.Request#SetBody` function.
func (r *Request) SetBody(reader io.Reader) {
	r.Request.Body = ioutil.NopCloser(reader)
}

// FormValue implements `whttp.Request#FormValue` function.
func (r *Request) FormValue(name string) string {
	return r.Request.FormValue(name)
}

// FormParams implements `whttp.Request#FormParams` function.
func (r *Request) FormParams() map[string][]string {
	if strings.HasPrefix(r.header.Get("Content-Type"), "multipart/form-data") {
		if err := r.ParseMultipartForm(defaultMemory); err != nil {
			//r.logger.Error(err)
		}
	} else {
		if err := r.ParseForm(); err != nil {
			//r.logger.Error(err)
		}
	}
	return map[string][]string(r.Request.Form)
}

// FormFile implements `whttp.Request#FormFile` function.
func (r *Request) FormFile(name string) (*multipart.FileHeader, error) {
	_, fh, err := r.Request.FormFile(name)
	return fh, err
}

// MultipartForm implements `whttp.Request#MultipartForm` function.
func (r *Request) MultipartForm() (*multipart.Form, error) {
	err := r.ParseMultipartForm(defaultMemory)
	return r.Request.MultipartForm, err
}

// Cookie implements `whttp.Request#Cookie` function.
func (r *Request) Cookie(name string) (server.Cookie, error) {
	c, err := r.Request.Cookie(name)
	if err != nil {
		//return nil, wgo.ErrCookieNotFound
		return nil, errors.New("err cookie not found")
	}
	return &Cookie{c}, nil
}

// Cookies implements `whttp.Request#Cookies` function.
func (r *Request) Cookies() []server.Cookie {
	cs := r.Request.Cookies()
	cookies := make([]server.Cookie, len(cs))
	for i, c := range cs {
		cookies[i] = &Cookie{c}
	}
	return cookies
}

func (r *Request) reset(req *http.Request, h *Header, u *URL) {
	r.Request = req
	r.header = h
	r.url = u
}
