package middlewares

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"

	"wgo"
	"wgo/server"
	"wgo/whttp"
	"wgo/whttp/fasthttp"
	"wgo/whttp/standard"
)

type (
	ReverseProxy struct {
		*httputil.ReverseProxy
		target *url.URL
	}
	netHTTPBody struct {
		b []byte
	}
)

func (r *netHTTPBody) Read(p []byte) (int, error) {
	if len(r.b) == 0 {
		return 0, io.EOF
	}
	n := copy(p, r.b)
	r.b = r.b[n:]
	return n, nil
}

func (r *netHTTPBody) Close() error {
	r.b = r.b[:0]
	return nil
}

const (
	CFG_KEY_PROXY = "proxy"
)

var Transport = &http.Transport{
	TLSClientConfig: &tls.Config{InsecureSkipVerify: true},

	DialContext: (&net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
		DualStack: true,
	}).DialContext,
	MaxIdleConns:          100,
	IdleConnTimeout:       90 * time.Second,
	TLSHandshakeTimeout:   10 * time.Second,
	ExpectContinueTimeout: 1 * time.Second,
}

func Proxy() wgo.MiddlewareFunc {
	pool := sync.Pool{
		New: func() interface{} {
			return &ReverseProxy{
				ReverseProxy: &httputil.ReverseProxy{
					Transport:      http.RoundTripper(Transport), // 连接参数
					ModifyResponse: nil,                          // 对返回信息进行修改
				},
			}
		},
	}
	return func(next wgo.HandlerFunc) wgo.HandlerFunc {
		return func(c *wgo.Context) (err error) {
			//return next(c)

			switch c.ServerMode() { // http才需要proxy
			case "wrpc", "rpc", "grpc":
				return next(c)
			}

			cfg := wgo.Cfg()
			if cfg.Get(CFG_KEY_PROXY) == nil { // 没有配置, 则跳过
				c.Info("not found proxy config")
				return next(c)
			}

			var config map[string](map[string]([]string))
			if err := cfg.UnmarshalKey(CFG_KEY_PROXY, &config); err != nil {
				c.Info("cfg.UnmarshalKey failed: %s", err)
				return next(c)
			}

			//req := c.Request().(*std.Request)
			//responseWriter := c.Response().(*std.Response)

			path := c.Path()
			config2, ok := config[c.Host()]
			if !ok {
				config2, _ = config["*"]
			}
			addrs, ok := config2[path]
			if !ok {
				addrs, _ = config2["*"]
			}

			if len(addrs) < 1 {
				return next(c)
			}

			// random select
			idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(addrs))))
			proxyUrl, err := url.Parse(addrs[int(idx.Int64())])

			c.Info("url: %v", proxyUrl)
			if err != nil {
				c.Error("proxyUrl error: %s\n", err)
				return nil
			}

			proxy := pool.Get().(*ReverseProxy)
			defer pool.Put(proxy)
			proxy.reset(proxyUrl)

			return proxy.doProxy(c)
		}
	}
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}
func copyFastHeader(dst server.Header, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Set(k, v) // 防止header重复
		}
	}
}

// skip header
var skipHeaders = []string{
	"Date",           // date以本机为准
	"Server",         // server以本机为准
	"Content-Length", // 内容长度以本机为准(也许经过再压缩)
}

// Hop-by-hop headers. These are removed when sent to the backend.
// http://www.w3.org/Protocols/rfc2616/rfc2616-sec13.html
var hopHeaders = []string{
	"Connection",
	"Proxy-Connection", // non-standard but still sent by libcurl and rejected by e.g. google
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"Te",      // canonicalized version of "TE"
	"Trailer", // not Trailers per URL above; http://www.rfc-editor.org/errata_search.php?eid=4522
	"Transfer-Encoding",
	"Upgrade",
}

// proxy handler
func (rp *ReverseProxy) doProxy(c *wgo.Context) error {

	var std bool
	var req *http.Request
	ctx := c.Context()
	//if cn, ok := rw.(http.CloseNotifier); ok {
	if cn, ok := c.Response().(http.CloseNotifier); ok { // 只有standard http实现了http.CloseNotifier
		// standard下直接拿到http.Requst
		std = true
		req = c.Request().(*standard.Request).Request
		var cancel context.CancelFunc
		ctx, cancel = context.WithCancel(ctx)
		defer cancel()
		notifyChan := cn.CloseNotify()
		go func() {
			select {
			case <-notifyChan:
				cancel()
			case <-ctx.Done():
			}
		}()
	} else {
		// fasthttp需要拼一个http.Requst, copied from github.com/valyala/fasthttp/fasthttpadaptor/adaptor.go#NewFastHTTPHandler
		var r http.Request

		fctx := c.Request().(*fasthttp.Request).RequestCtx

		body := fctx.PostBody()
		r.Method = string(fctx.Method())
		r.Proto = "HTTP/1.1"
		r.ProtoMajor = 1
		r.ProtoMinor = 1
		r.RequestURI = string(fctx.RequestURI())
		r.ContentLength = int64(len(body))
		r.Host = string(fctx.Host())
		r.RemoteAddr = fctx.RemoteAddr().String()

		hdr := make(http.Header)
		fctx.Request.Header.VisitAll(func(k, v []byte) {
			sk := string(k)
			sv := string(v)
			switch sk {
			case "Transfer-Encoding":
				r.TransferEncoding = append(r.TransferEncoding, sv)
			default:
				hdr.Set(sk, sv)
			}
		})
		r.Header = hdr
		r.Body = &netHTTPBody{body}
		rURL, _ := url.ParseRequestURI(r.RequestURI)
		//if err != nil {
		//	return err
		//}
		r.URL = rURL

		req = &r
	}

	outreq := new(http.Request)
	*outreq = *req // includes shallow copies of maps, but okay
	if req.ContentLength == 0 {
		outreq.Body = nil // Issue 16036: nil Body for http.Transport retries
	}
	outreq = outreq.WithContext(ctx)

	rp.Director(outreq)
	outreq.Close = false

	// We are modifying the same underlying map from req (shallow
	// copied above) so we only copy it if necessary.
	copiedHeaders := false

	// Remove hop-by-hop headers listed in the "Connection" header.
	// See RFC 2616, section 14.10.
	if conn := outreq.Header.Get("Connection"); conn != "" {
		for _, f := range strings.Split(conn, ",") {
			if f = strings.TrimSpace(f); f != "" {
				if !copiedHeaders {
					outreq.Header = make(http.Header)
					copyHeader(outreq.Header, req.Header)
					copiedHeaders = true
				}
				outreq.Header.Del(f)
			}
		}
	}

	// Remove hop-by-hop headers to the backend. Especially
	// important is "Connection" because we want a persistent
	// connection, regardless of what the client sent to us.
	for _, h := range hopHeaders {
		if outreq.Header.Get(h) != "" {
			if !copiedHeaders {
				outreq.Header = make(http.Header)
				copyHeader(outreq.Header, req.Header)
				copiedHeaders = true
			}
			outreq.Header.Del(h)
		}
	}

	// add request-id and depth
	if rid := c.RequestID(); rid != "" {
		outreq.Header.Set(whttp.HeaderXRequestId, rid)
		outreq.Header.Set(whttp.HeaderXDepth, fmt.Sprint(c.Depth()))
	}

	// add x-forwarded-for
	if clientIP := c.ClientIP(); clientIP != "" {
		// If we aren't the first proxy retain prior
		// X-Forwarded-For information as a comma+space
		// separated list and fold multiple headers into one.
		if prior, ok := outreq.Header["X-Forwarded-For"]; ok {
			clientIP = strings.Join(prior, ", ") + ", " + clientIP
		}
		outreq.Header.Set("X-Forwarded-For", clientIP)
	}

	res, err := rp.Transport.RoundTrip(outreq)
	if err != nil {
		return err
	}

	// Remove hop-by-hop headers listed in the
	// "Connection" header of the response.
	if conn := res.Header.Get("Connection"); conn != "" {
		for _, f := range strings.Split(conn, ",") {
			if f = strings.TrimSpace(f); f != "" {
				res.Header.Del(f)
			}
		}
	}

	for _, h := range hopHeaders {
		res.Header.Del(h)
	}

	for _, h := range skipHeaders {
		res.Header.Del(h)
	}

	if rp.ModifyResponse != nil {
		if err := rp.ModifyResponse(res); err != nil {
			c.Response().(whttp.Response).WriteHeader(http.StatusBadGateway)
			return err
		}
	}

	// copy content to wgo
	if std {
		rw := c.Response().(*standard.Response).ResponseWriter

		copyHeader(rw.Header(), res.Header)

		// The "Trailer" header isn't included in the Transport's response,
		// at least for *http.Transport. Build it up from Trailer.
		if len(res.Trailer) > 0 {
			var trailerKeys []string
			for k := range res.Trailer {
				trailerKeys = append(trailerKeys, k)
			}
			rw.Header().Add("Trailer", strings.Join(trailerKeys, ", "))
		}

		rw.WriteHeader(res.StatusCode)
		if len(res.Trailer) > 0 {
			// Force chunking if we saw a response trailer.
			// This prevents net/http from calculating the length for short
			// bodies and adding a Content-Length.
			if fl, ok := rw.(http.Flusher); ok {
				fl.Flush()
			}
		}
		rp.copyResponse(rw, res.Body)
		res.Body.Close() // close now, instead of defer, to populate res.Trailer
		copyHeader(rw.Header(), res.Trailer)
	} else { // fasthttp
		copyFastHeader(c.Response().(whttp.Response).Header(), res.Header)
		c.Response().(whttp.Response).WriteHeader(res.StatusCode)

		// The "Trailer" header isn't included in the Transport's response,
		// at least for *http.Transport. Build it up from Trailer.
		if len(res.Trailer) > 0 {
			var trailerKeys []string
			for k := range res.Trailer {
				trailerKeys = append(trailerKeys, k)
			}
			c.Response().(whttp.Response).Header().Add("Trailer", strings.Join(trailerKeys, ", "))
		}

		if content, err := ioutil.ReadAll(res.Body); err != nil {
			return err
		} else {
			c.Response().(whttp.Response).Write(content)
		}
		res.Body.Close()
	}
	return nil
}

// proxy reset
func (rp *ReverseProxy) reset(target *url.URL) {
	targetQuery := target.RawQuery
	director := func(req *http.Request) {
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.URL.Path = singleJoiningSlash(target.Path, req.URL.Path)
		if targetQuery == "" || req.URL.RawQuery == "" {
			req.URL.RawQuery = targetQuery + req.URL.RawQuery
		} else {
			req.URL.RawQuery = targetQuery + "&" + req.URL.RawQuery
		}
		if _, ok := req.Header["User-Agent"]; !ok {
			// explicitly disable User-Agent so it's not set to default value
			req.Header.Set("User-Agent", "")
		}
	}
	rp.Director = director
}

func (rp *ReverseProxy) copyResponse(dst io.Writer, src io.Reader) {
	p := rp.ReverseProxy
	if p.FlushInterval != 0 {
		if wf, ok := dst.(writeFlusher); ok {
			mlw := &maxLatencyWriter{
				dst:     wf,
				latency: p.FlushInterval,
				done:    make(chan bool),
			}
			go mlw.flushLoop()
			defer mlw.stop()
			dst = mlw
		}
	}

	var buf []byte
	if p.BufferPool != nil {
		buf = p.BufferPool.Get()
	}
	io.CopyBuffer(dst, src, buf)
	if p.BufferPool != nil {
		p.BufferPool.Put(buf)
	}
}

type writeFlusher interface {
	io.Writer
	http.Flusher
}
type maxLatencyWriter struct {
	dst     writeFlusher
	latency time.Duration

	lk   sync.Mutex // protects Write + Flush
	done chan bool
}

func (m *maxLatencyWriter) Write(p []byte) (int, error) {
	m.lk.Lock()
	defer m.lk.Unlock()
	return m.dst.Write(p)
}

func (m *maxLatencyWriter) flushLoop() {
	t := time.NewTicker(m.latency)
	defer t.Stop()
	for {
		select {
		case <-m.done:
			if onExitFlushLoop != nil {
				onExitFlushLoop()
			}
			return
		case <-t.C:
			m.lk.Lock()
			m.dst.Flush()
			m.lk.Unlock()
		}
	}
}

func (m *maxLatencyWriter) stop() { m.done <- true }

// onExitFlushLoop is a callback set by tests to detect the state of the
// flushLoop() goroutine.
var onExitFlushLoop func()

func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}
	return a + b
}
