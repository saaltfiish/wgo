package wgo

import (
	"encoding/json"
	"encoding/xml"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"

	ctx "golang.org/x/net/context"

	"wgo/server"
	"wgo/whttp"
)

func (c *Context) RequestBody() io.Reader {
	return c.Request().(whttp.Request).Body()
}

func (c *Context) Path() string {
	return c.path
}

func (c *Context) SetPath(p string) {
	c.path = p
}

func (c *Context) P(i int) (value string) {
	l := len(c.pnames)
	if i < l {
		value = c.pvalues[i]
	}
	return
}

func (c *Context) QueryParam(name string) string {
	return c.request.(whttp.Request).URL().QueryParam(name)
}

func (c *Context) QueryParams() map[string][]string {
	return c.request.(whttp.Request).URL().QueryParams()
}

func (c *Context) FormValue(name string) string {
	return c.request.(whttp.Request).FormValue(name)
}

func (c *Context) FormParams() map[string][]string {
	return c.request.(whttp.Request).FormParams()
}

func (c *Context) FormFile(name string) (*multipart.FileHeader, error) {
	return c.request.(whttp.Request).FormFile(name)
}

func (c *Context) MultipartForm() (*multipart.Form, error) {
	return c.request.(whttp.Request).MultipartForm()
}

func (c *Context) Cookie(name string) (server.Cookie, error) {
	return c.request.(whttp.Request).Cookie(name)
}

func (c *Context) SetCookie(cookie server.Cookie) {
	c.response.(whttp.Response).SetCookie(cookie)
}

func (c *Context) Cookies() []server.Cookie { // interface{} = []http.Cookie
	return c.request.(whttp.Request).Cookies()
}

func (c *Context) Set(key string, val interface{}) {
	c.context = ctx.WithValue(c.context, key, val)
}

func (c *Context) Get(key string) interface{} {
	return c.context.Value(key)
}

func (c *Context) Bind(i interface{}) error {
	return c.mux.(*whttp.Mux).Binder().Bind(i, c.request.(whttp.Request))
}

func (c *Context) File(file string) error {
	f, err := os.Open(file)
	if err != nil {
		return whttp.ErrNotFound
	}
	defer f.Close()

	fi, _ := f.Stat()
	if fi.IsDir() {
		file = filepath.Join(file, "index.html")
		f, err = os.Open(file)
		if err != nil {
			return whttp.ErrNotFound
		}
		if fi, err = f.Stat(); err != nil {
			return err
		}
	}
	return c.ServeContent(f, fi.Name(), fi.ModTime())
}

func (c *Context) Attachment(r io.ReadSeeker, name string) (err error) {
	c.response.(whttp.Response).Header().Set(whttp.HeaderContentType, ContentTypeByExtension(name))
	c.response.(whttp.Response).Header().Set(whttp.HeaderContentDisposition, "attachment; filename="+name)
	c.response.(whttp.Response).WriteHeader(http.StatusOK)
	_, err = io.Copy(c.response.(whttp.Response), r)
	return
}

func (c *Context) NoContent(code int) error {
	c.response.(whttp.Response).WriteHeader(code)
	return nil
}

func (c *Context) Redirect(code int, url string) error {
	if code < http.StatusMultipleChoices || code > http.StatusTemporaryRedirect {
		return whttp.ErrInvalidRedirectCode
	}
	c.response.(whttp.Response).Header().Set(whttp.HeaderLocation, url)
	c.response.(whttp.Response).WriteHeader(code)
	return nil
}

//func (c *Context) Mux() *whttp.Mux {
//	return c.mux.(*whttp.Mux)
//}

func (c *Context) ServeContent(content io.ReadSeeker, name string, modtime time.Time) error {
	req := c.Request().(whttp.Request)
	res := c.Response().(whttp.Response)

	if t, err := time.Parse(http.TimeFormat, req.Header().Get(whttp.HeaderIfModifiedSince)); err == nil && modtime.Before(t.Add(1*time.Second)) {
		res.Header().Del(whttp.HeaderContentType)
		res.Header().Del(whttp.HeaderContentLength)
		return c.NoContent(http.StatusNotModified)
	}

	res.Header().Set(whttp.HeaderContentType, ContentTypeByExtension(name))
	res.Header().Set(whttp.HeaderLastModified, modtime.UTC().Format(http.TimeFormat))
	res.WriteHeader(http.StatusOK)
	_, err := io.Copy(res, content)
	return err
}

// ContentTypeByExtension returns the MIME type associated with the file based on
// its extension. It returns `application/octet-stream` incase MIME type is not
// found.
func ContentTypeByExtension(name string) (t string) {
	if t = mime.TypeByExtension(filepath.Ext(name)); t == "" {
		t = whttp.MIMEOctetStream
	}
	Info("name: %s, ext: %s, type: %s", name, filepath.Ext(name), t)
	return
}

func (c *Context) HTTPReset(req whttp.Request, res whttp.Response) {
	c.context = ctx.Background()
	c.request = req
	c.response = res
	c.mode = "http"
	c.start = time.Now()
	c.access.Reset(c.start)
	c.auth = false
	c.encoding = ""
	c.node = nil
	c.reqID = ""
	c.noCache = false
	c.ext = nil
}

func (c *Context) HTML(code int, html string) (err error) {
	return c.HTMLBlob(code, []byte(html))
}

func (c *Context) HTMLBlob(code int, b []byte) (err error) {
	return c.Blob(code, whttp.MIMETextHTMLCharsetUTF8, b)
}

func (c *Context) String(code int, s string) (err error) {
	return c.Blob(code, whttp.MIMETextPlainCharsetUTF8, []byte(s))
}

func (c *Context) JSON(code int, i interface{}) (err error) {
	if debug {
		return c.JSONPretty(code, i, "  ")
	}
	b, err := json.Marshal(i)
	if err != nil {
		return
	}
	return c.JSONBlob(code, b)
}

func (c *Context) JSONPretty(code int, i interface{}, indent string) (err error) {
	b, err := json.MarshalIndent(i, "", indent)
	if err != nil {
		return
	}
	return c.JSONBlob(code, b)
}

func (c *Context) JSONBlob(code int, b []byte) (err error) {
	return c.Blob(code, whttp.MIMEApplicationJSONCharsetUTF8, b)
}

func (c *Context) JSONP(code int, callback string, i interface{}) (err error) {
	b, err := json.Marshal(i)
	if err != nil {
		return
	}
	return c.JSONPBlob(code, callback, b)
}

func (c *Context) JSONPBlob(code int, callback string, b []byte) (err error) {
	c.response.(whttp.Response).Header().Set(whttp.HeaderContentType, whttp.MIMEApplicationJavaScriptCharsetUTF8)
	c.response.(whttp.Response).WriteHeader(code)
	if _, err = c.response.(whttp.Response).Write([]byte(callback + "(")); err != nil {
		return
	}
	if _, err = c.response.(whttp.Response).Write(b); err != nil {
		return
	}
	_, err = c.response.(whttp.Response).Write([]byte(");"))
	return
}

func (c *Context) XML(code int, i interface{}) (err error) {
	if debug {
		return c.XMLPretty(code, i, "  ")
	}
	b, err := xml.Marshal(i)
	if err != nil {
		return
	}
	return c.XMLBlob(code, b)
}

func (c *Context) XMLPretty(code int, i interface{}, indent string) (err error) {
	b, err := xml.MarshalIndent(i, "", indent)
	if err != nil {
		return
	}
	return c.XMLBlob(code, b)
}

func (c *Context) XMLBlob(code int, b []byte) (err error) {
	c.response.(whttp.Response).Header().Set(whttp.HeaderContentType, whttp.MIMEApplicationXMLCharsetUTF8)
	c.response.(whttp.Response).WriteHeader(code)
	if _, err = c.response.(whttp.Response).Write([]byte(xml.Header)); err != nil {
		return
	}
	_, err = c.response.(whttp.Response).Write(b)
	return
}

func (c *Context) Blob(code int, contentType string, b []byte) (err error) {
	c.response.(whttp.Response).Header().Set(whttp.HeaderContentType, contentType)
	c.response.(whttp.Response).WriteHeader(code)
	if c.Encoding() == "gzip" && len(b) > 200 {
		// gzip headers
		c.response.(whttp.Response).Header().Set(whttp.HeaderVary, whttp.HeaderAcceptEncoding)

		if _, err = c.response.(whttp.Response).WriteGzip(b); err == nil {
			c.response.(whttp.Response).Header().Set(whttp.HeaderContentEncoding, "gzip")
		}
	} else {
		_, err = c.response.(whttp.Response).Write(b)
	}
	return
}

func (c *Context) Stream(code int, contentType string, r io.Reader) (err error) {
	c.response.(whttp.Response).Header().Set(whttp.HeaderContentType, contentType)
	c.response.(whttp.Response).WriteHeader(code)
	_, err = io.Copy(c.response.(whttp.Response), r)
	return
}

// options
func (c *Context) Options(key string) interface{} {
	if op := c.Node(); op != nil { // 路由配置
		if opts, ok := op.(*whttp.RouteNode).Opts[key]; ok {
			return opts
		}
	}
	return nil
}
