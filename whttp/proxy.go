package whttp

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"

	wcache "wgo/cache"
	"wgo/environ"
	"wgo/server"
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
		Timeout:   90 * time.Second,
		KeepAlive: 90 * time.Second,
		DualStack: true,
	}).DialContext,
	MaxIdleConns:          100,
	IdleConnTimeout:       90 * time.Second,
	TLSHandshakeTimeout:   10 * time.Second,
	ExpectContinueTimeout: 1 * time.Second,
}

func Proxy() MiddlewareFunc {
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
	cache := wcache.NewCache()
	var config map[string](map[string]interface{})
	if err := environ.Cfg().UnmarshalKey(CFG_KEY_PROXY, &config); err != nil {
		Info("not found proxy config")
	}
	return func(next HandlerFunc) HandlerFunc {
		return func(c Context) (err error) {

			if config == nil { // 没有proxy配置, 跳过
				return next(c)
			}

			switch c.ServerMode() { // http才需要proxy
			case "wrpc", "rpc", "grpc":
				return next(c)
			}

			// 可针对endpoint进行proxy设置
			endpoint := "/"
			path := c.Request().(Request).URL().Path()
			if pp := strings.SplitN(path, "/", 3); len(pp) >= 2 {
				endpoint += pp[1]
			}
			//c.Info("path: %s, endpoint: %s", path, endpoint)
			// 第一层配置域名, `*`为通配符
			config2, ok := config[c.Host()]
			if !ok {
				config2, _ = config["*"]
			}
			proxyCfg, ok := config2[endpoint]
			if !ok {
				proxyCfg, _ = config2["*"]
			}

			var proxyUrl *url.URL
			var cacheOpts Options
			if addrs, ok := proxyCfg.([]interface{}); ok && len(addrs) > 0 { // 旧配置, 只配置地址
				// random select
				proxyUrl, _ = randomAddr(addrs)
			} else if cc, ok := proxyCfg.(map[string]interface{}); ok { // 新配置, 可配置缓存
				if addrs, ok := cc["addrs"].([]interface{}); ok && len(addrs) > 0 {
					proxyUrl, _ = randomAddr(addrs)
					cacheOpts = cc
				}
			} else {
				c.Error("config wrong, host: %s, cfg: %q, path: %s", c.Host(), proxyCfg, path)
			}

			if proxyUrl == nil {
				c.Info("not foud proxy for you")
				return next(c)
			}
			// c.Info("host: %s, cfg: %q, path: %s, proxyUrl: %s", c.Host(), proxyCfg, path, proxyUrl)

			ttl := 0
			key := ""
			if cacheOpts != nil { // 缓存!
				if ttlf := cacheOpts["ttl"].(float64); ttlf > 0 {
					ttl = int(ttlf)
				}
				paramString := ""
				if params, ok := cacheOpts["params"].([]interface{}); ok && len(params) > 0 { // 需要缓存的query参数
					ps := bytes.Buffer{}
					for _, n := range params {
						ps.WriteString(c.QueryParam(n.(string)) + ",")
					}
					paramString = ps.String()
				}
				headerString := ""
				if headers, ok := cacheOpts["headers"].([]interface{}); ok && len(headers) > 0 { // 需要缓存的header参数
					hs := bytes.Buffer{}
					for _, n := range headers {
						hs.WriteString(c.Request().(Request).Header().Get(n.(string)))
					}
					headerString = hs.String()
				}
				key = fmt.Sprintf("%s:%s:%s:%s:%s:%s", c.Request().(Request).Method(), c.Request().(Request).URL().Path(), c.Mux().Name(), c.Encoding(), paramString, headerString) // 缓存决定因素为method,path,engine,encoding,params,headers
				if res, err := cache.Get([]byte(key)); err == nil {
					c.Info("[proxy] got key: %s", key)
					res.(Response).CopyTo(c.Response())
					return nil
				} else {
					c.Error("get key(%s) failed: %s", key, err.Error())
				}
			}

			proxy := pool.Get().(*ReverseProxy)
			defer pool.Put(proxy)
			proxy.reset(proxyUrl)

			err = proxy.doProxy(c)
			if err == nil && cacheOpts != nil {
				// success, cache response
				nr := c.Mux().(*Mux).NewResponse()
				c.Response().(Response).CopyTo(nr)
				cache.Set([]byte(key), nr, ttl)
				c.Warn("[proxy] response(%s) cached", key)
			}
			return err
		}
	}
}

func randomAddr(addrs []interface{}) (addr *url.URL, err error) {
	idx, _ := rand.Int(rand.Reader, big.NewInt(int64(len(addrs))))
	return url.Parse(addrs[int(idx.Int64())].(string))
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
func (rp *ReverseProxy) doProxy(c Context) error {

	//var std bool
	var req *http.Request
	ctx := c.Context()
	//if cn, ok := rw.(http.CloseNotifier); ok {
	if cn, ok := c.Response().(http.CloseNotifier); ok {
		// 只有standard http实现了http.CloseNotifier, 因此本质上只有standard能做proxy
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
		//fasthttp需要拼一个http.Requst,copied from github.com/valyala/fasthttp/fasthttpadaptor/adaptor.go#NewFastHTTPHandler
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
		r.URL = rURL

		req = &r
	}

	outreq := new(http.Request)
	*outreq = *req // includes shallow copies of maps, but okay
	if req.ContentLength == 0 {
		outreq.Body = nil // Issue 16036: nil Body for http.Transport retries
	}
	outreq = outreq.WithContext(ctx)
	if req.ContentLength == 0 {
		outreq.Body = nil // Issue 16036: nil Body for http.Transport retries
	}

	rp.Director(outreq)
	outreq.Close = true

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
		hv := outreq.Header.Get(h)
		if hv == "" {
			continue
		}
		if h == "Te" && hv == "trailers" {
			// Issue 21096: tell backend applications that
			// care about trailer support that we support
			// trailers. (We do, but we don't go out of
			// our way to advertise that unless the
			// incoming client request thought it was
			// worth mentioning)
			continue
		}
		if !copiedHeaders {
			outreq.Header = make(http.Header)
			copyHeader(outreq.Header, req.Header)
			copiedHeaders = true
		}
		outreq.Header.Del(h)
		outreq.Header.Del(h)
	}

	// add request-id and depth
	if rid := c.RequestID(); rid != "" {
		outreq.Header.Set(HeaderXRequestId, rid)
		outreq.Header.Set(HeaderXDepth, fmt.Sprint(c.Depth()))
	}

	// add x-forwarded-for
	if clientIP := c.ClientIP(); clientIP != "" {
		// If we aren't the first proxy retain prior
		// X-Forwarded-For information as a comma+space
		// separated list and fold multiple headers into one.
		if prior, ok := outreq.Header[HeaderXForwardedFor]; ok {
			clientIP = strings.Join(prior, ", ") + ", " + clientIP
		}
		outreq.Header.Set(HeaderXForwardedFor, clientIP)
	}

	// add X-Forwarded-Proto
	// c.Error("add X-Forwarded-Proto: %s", c.Scheme())
	outreq.Header.Set(HeaderXForwardedProto, c.Scheme())

	res, err := rp.Transport.RoundTrip(outreq)
	if err != nil {
		c.Error("RoundTrip error: %s", err)
		return err
	}
	// del content-length
	res.Header.Del(HeaderContentLength)

	// if _, ok := c.Response().(http.CloseNotifier); !ok {
	// 	// fasthttp
	// 	r := c.Response().(*fasthttp.Response)
	// 	r.RequestCtx.Response.SetBodyStream(bytes.NewReader(r.RequestCtx.Response.Body()), -1)
	// }

	// cors header(Access-Control-Allow-*)
	if origin := outreq.Header.Get(HeaderOrigin); origin != "" {
		res.Header.Set(HeaderAccessControlAllowOrigin, origin)
		res.Header.Set(HeaderAccessControlAllowCredentials, "true")
	}
	if c.Request().(Request).Method() == "OPTIONS" { // 统一处理options请求
		res.Header.Set(HeaderAccessControlMaxAge, "86400")
		res.Header.Set(HeaderAccessControlAllowMethods, "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		if ch := outreq.Header.Get(HeaderAccessControlRequestHeaders); ch != "" {
			res.Header.Set(HeaderAccessControlAllowHeaders, ch) // 来者不拒
		}
		c.Response().(Response).WriteHeader(StatusOK)
		copyFastHeader(c.Response().(Response).Header(), res.Header)
		return nil
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
			c.Response().(Response).WriteHeader(http.StatusBadGateway)
			return err
		}
	}
	c.Response().(Response).WriteHeader(res.StatusCode)

	// copy content to wgo
	//rw := c.Response().(*standard.Response).ResponseWriter
	rw := c.Response().(Response).Writer()

	//copyHeader(rw.Header(), res.Header)
	copyFastHeader(c.Response().(Response).Header(), res.Header)

	// The "Trailer" header isn't included in the Transport's response,
	// at least for *http.Transport. Build it up from Trailer.
	if len(res.Trailer) > 0 {
		var trailerKeys []string
		for k := range res.Trailer {
			trailerKeys = append(trailerKeys, k)
		}
		//rw.Header().Add("Trailer", strings.Join(trailerKeys, ", "))
		c.Response().(Response).Header().Add("Trailer", strings.Join(trailerKeys, ", "))
	}

	//rw.WriteHeader(res.StatusCode)
	if len(res.Trailer) > 0 {
		// Force chunking if we saw a response trailer.
		// This prevents net/http from calculating the length for short
		// bodies and adding a Content-Length.
		if fl, ok := rw.(http.Flusher); ok {
			fl.Flush()
		}
	}
	rp.copyResponse(c.Response().(Response), res.Body, rp.flushInterval(res))
	res.Body.Close() // close now, instead of defer, to populate res.Trailer
	// Debug("[doProxy]size: %d", c.Response().(Response).Size())
	//copyHeader(rw.Header(), res.Trailer)
	copyFastHeader(c.Response().(Response).Header(), res.Trailer)
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

// flushInterval returns the p.FlushInterval value, conditionally
// overriding its value for a specific request/response.
func (rp *ReverseProxy) flushInterval(res *http.Response) time.Duration {
	resCT := res.Header.Get("Content-Type")

	// For Server-Sent Events responses, flush immediately.
	// The MIME type is defined in https://www.w3.org/TR/eventsource/#text-event-stream
	if resCT == MIMEEventStream {
		Debug("[flushInterval]Server-Sent Events responses: %s", resCT)
		return -1 // negative means immediately
	}

	// TODO: more specific cases? e.g. res.ContentLength == -1?
	return rp.ReverseProxy.FlushInterval
}

func (rp *ReverseProxy) copyResponse(dst io.Writer, src io.Reader, flushInterval time.Duration) error {
	p := rp.ReverseProxy
	if flushInterval != 0 {
		Debug("[copyResponse]flushInterval: %d", flushInterval)
		if wf, ok := dst.(writeFlusher); ok {
			Debug("[copyResponse]got writeFlusher")
			mlw := &maxLatencyWriter{
				dst:     wf,
				latency: flushInterval,
				// done:    make(chan bool),
			}
			// go mlw.flushLoop()
			defer mlw.stop()
			dst = mlw
		}
	}

	var buf []byte
	if p.BufferPool != nil {
		buf = p.BufferPool.Get()
		defer p.BufferPool.Put(buf)
	}
	// io.CopyBuffer(dst, src, buf)
	_, err := rp.copyBuffer(dst, src, buf)
	return err
}

// copyBuffer returns any write errors or non-EOF read errors, and the amount
// of bytes written.
func (p *ReverseProxy) copyBuffer(dst io.Writer, src io.Reader, buf []byte) (int64, error) {
	if len(buf) == 0 {
		buf = make([]byte, 32*1024)
	}
	var written int64
	for {
		nr, rerr := src.Read(buf)
		if rerr != nil && rerr != io.EOF && rerr != context.Canceled {
			Debug("httputil: ReverseProxy read error during body copy: %v", rerr)
		}
		if nr > 0 {
			nw, werr := dst.Write(buf[:nr])
			Debug("[copyBuffer]nw: %d", nw)
			if nw > 0 {
				written += int64(nw)
			}
			if werr != nil {
				return written, werr
			}
			if nr != nw {
				return written, io.ErrShortWrite
			}
		}
		if rerr != nil {
			if rerr == io.EOF {
				rerr = nil
			}
			return written, rerr
		}
	}
}

type writeFlusher interface {
	io.Writer
	http.Flusher
}
type maxLatencyWriter struct {
	dst     writeFlusher
	latency time.Duration

	mu           sync.Mutex // protects Write + Flush
	t            *time.Timer
	flushPending bool
}

func (m *maxLatencyWriter) Write(p []byte) (n int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	n, err = m.dst.Write(p)
	if m.latency < 0 {
		Debug("[Write]m.latency: %d", m.latency)
		m.dst.Flush()
		return
	}
	if m.flushPending {
		return
	}
	if m.t == nil {
		m.t = time.AfterFunc(m.latency, m.delayedFlush)
	} else {
		m.t.Reset(m.latency)
	}
	m.flushPending = true
	return
}

func (m *maxLatencyWriter) delayedFlush() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.dst.Flush()
	m.flushPending = false
}

func (m *maxLatencyWriter) stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.t != nil {
		m.t.Stop()
	}
}

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
