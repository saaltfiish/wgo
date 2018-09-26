package wgo

import (
	"net"
	"strconv"
	"strings"
	"time"

	ctx "golang.org/x/net/context"

	"wgo/environ"
	"wgo/server"
	"wgo/whttp"
	"wgo/wrpc"
)

type (
	Context struct {
		context  ctx.Context
		mux      server.Mux
		request  interface{} // 具体的request在各自包内定义
		response interface{} // 具体的response在各自包内定义
		job      *Job        // job
		auth     bool
		encoding string
		node     interface{} // router node
		path     string
		pnames   []string
		pvalues  []string
		handler  HandlerFunc
		start    time.Time
		reqID    string // request id
		logger   server.Logger
		mode     string
		access   *AccessLog
		noCache  bool
		// ext      interface{} // 额外信息
	}
)

// Generator
func NewContext() interface{} {
	c := &Context{
		access: NewAccessLog(),
	}
	c.SetLogger(Logger())
	return c
}

// clone
func (c *Context) Clone() *Context {
	nc := &Context{}
	nc.context = ctx.Background()
	nc.mode = "job"
	nc.start = time.Now()
	nc.access = c.Access().Clone()
	nc.reqID = c.RequestID()
	nc.logger = c.logger
	return nc
}

func (c *Context) Access() *AccessLog {
	return c.access
}

func (c *Context) Context() ctx.Context {
	return c.context
}

func (c *Context) SetContext(ctx ctx.Context) {
	c.context = ctx
}

func (c *Context) Deadline() (time.Time, bool) {
	return c.context.Deadline()
}

func (c *Context) Done() <-chan struct{} {
	return c.context.Done()
}

func (c *Context) Err() error {
	return c.context.Err()
}

func (c *Context) Set(key string, val interface{}) {
	c.context = ctx.WithValue(c.context, key, val)
}

func (c *Context) Get(key string) interface{} {
	return c.context.Value(key)
}

func (c *Context) Value(key interface{}) interface{} {
	return c.context.Value(key)
}

func (c *Context) Job() *Job {
	return c.job
}

func (c *Context) Request() interface{} {
	return c.request
}

func (c *Context) Response() interface{} {
	return c.response
}

func (c *Context) Start() time.Time {
	return c.start
}

func (c *Context) Sub() time.Duration {
	return time.Now().Sub(c.start)
}

func (c *Context) SetExt(ext interface{}) {
	// c.ext = ext
	c.Set("__!ext!__", ext)
}

func (c *Context) Ext() interface{} {
	// return c.ext
	return c.Get("__!ext!__")
}

func (c *Context) SetRequestID(rid string) {
	c.reqID = rid
}

func (c *Context) RequestID() string {
	return c.reqID
}

// server mode
func (c *Context) ServerMode() string {
	return c.mode
}

// scheme
func (c *Context) Scheme() string {
	switch c.ServerMode() {
	case "http", "https", "whttp":
		return c.Request().(whttp.Request).Scheme()
	case "rpc", "wrpc", "grpc":
	default:
	}
	return ""
}

func (c *Context) Param(name string) (value string) {
	l := len(c.pnames)
	for i, n := range c.pnames {
		if n == name && i < l {
			value = c.pvalues[i]
			break
		}
	}
	return
}
func (c *Context) ParamNames() []string {
	return c.pnames
}

func (c *Context) SetParamNames(names ...string) {
	c.pnames = names
}

func (c *Context) ParamValues() []string {
	return c.pvalues
}

func (c *Context) SetParamValues(values ...string) {
	c.pvalues = values
}

func (c *Context) SetLogger(l interface{}) {
	c.logger = l.(server.Logger)
}

func (c *Context) Logger() server.Logger {
	return c.logger
}

// errors
func (c *Context) NewError(code int, msg string) *server.ServerError {
	return server.NewError(code, msg)
}
func (c *Context) NewErrorf(code int, format string, a ...interface{}) *server.ServerError {
	return server.NewErrorf(code, format, a...)
}
func (c *Context) ERROR(err error) {
	se := server.WrapError(err)
	switch c.ServerMode() {
	case "http", "https", "whttp":
		c.JSON(se.HTTPStatusCode(), se)
	case "rpc", "wrpc", "grpc":
		c.Response().(*wrpc.Response).Err = wrpc.Err(err)
	}
}

// mux
func (c *Context) SetMux(m server.Mux) {
	c.mux = m
}
func (c *Context) Mux() server.Mux {
	return c.mux
}

// client ip
func (c *Context) ClientIP() string {
	switch c.ServerMode() {
	case "http", "https", "whttp":
		if ip, _, err := net.SplitHostPort(strings.TrimSpace(c.Request().(whttp.Request).RemoteAddress())); err == nil {
			return ip
		}
	case "rpc", "wrpc", "grpc":
		if ip, _, err := net.SplitHostPort(strings.TrimSpace(c.Request().(*wrpc.Request).RemoteAddress())); err == nil {
			return ip
		}
	default:
	}
	return ""
}

// method
func (c *Context) Method() string {
	switch c.ServerMode() {
	case "http", "https", "whttp":
		return c.Request().(whttp.Request).Method()
	case "rpc", "wrpc", "grpc":
	default:
	}
	return ""
}

// query
func (c *Context) Query() string {
	switch c.ServerMode() {
	case "http", "https", "whttp":
		return c.Request().(whttp.Request).Method() + " " + c.Request().(whttp.Request).URL().Path()
	case "rpc", "wrpc", "grpc":
		return c.Request().(*wrpc.Request).Query()
	default:
	}
	return ""
}

// uri
func (c *Context) Params() string {
	switch c.ServerMode() {
	case "http", "https", "whttp":
		return c.Request().(whttp.Request).URL().QueryString()
	case "rpc", "wrpc", "grpc":
	default:
	}
	return ""
}

// host
func (c *Context) Host() string {
	switch c.ServerMode() {
	case "http", "https", "whttp":
		oh := c.Request().(whttp.Request).Host()
		if host, _, err := net.SplitHostPort(oh); err == nil {
			return host
		} else if oh != "" {
			// host不包含:,直接返回
			return oh
		}
	case "rpc", "wrpc", "grpc":
	default:
	}
	return ""
}

// origin
func (c *Context) Origin() string {
	switch c.ServerMode() {
	case "http", "https", "whttp":
		return c.RequestHeader().Get(whttp.HeaderOrigin)
	case "rpc", "wrpc", "grpc":
	default:
	}
	return ""
}

// user-agent
func (c *Context) UserAgent() string {
	switch c.ServerMode() {
	case "http", "https", "whttp":
		return c.Request().(whttp.Request).UserAgent()
	case "rpc", "wrpc", "grpc":
	default:
	}
	return ""
}
func (c *Context) Referer() string {
	switch c.ServerMode() {
	case "http", "https", "whttp":
		return c.Request().(whttp.Request).Referer()
	case "rpc", "wrpc", "grpc":
	default:
	}
	return ""
}

// req len
func (c *Context) ReqLen() int64 {
	switch c.ServerMode() {
	case "http", "https", "whttp":
		return c.Request().(whttp.Request).ContentLength()
	case "rpc", "wrpc", "grpc":
	default:
	}
	return 0
}
func (c *Context) RespLen() int64 {
	switch c.ServerMode() {
	case "http", "https", "whttp":
		return c.Response().(whttp.Response).Size()
	case "rpc", "wrpc", "grpc":
	default:
	}
	return 0
}
func (c *Context) Status() int {
	switch c.ServerMode() {
	case "http", "https", "whttp":
		return c.response.(whttp.Response).Status()
	case "rpc", "wrpc", "grpc":
	default:
	}
	return 0
}

// header
func (c *Context) RequestHeader() server.Header {
	switch c.ServerMode() {
	case "http", "https", "whttp":
		return c.request.(whttp.Request).Header()
	case "rpc", "wrpc", "grpc":
		return c.request.(*wrpc.Request).Header()
	default:
	}
	return nil
}
func (c *Context) ResponseHeader() server.Header {
	switch c.ServerMode() {
	case "http", "https", "whttp":
		return c.response.(whttp.Response).Header()
	case "rpc", "wrpc", "grpc":
		return c.response.(*wrpc.Response).Header()
	default:
	}
	return nil
}

// auth
func (c *Context) Authorize() { // 授权
	c.auth = true
}
func (c *Context) Authorized() bool { // 是否已授权
	return c.auth
}

// encoding
func (c *Context) Encoding() string {
	if c.encoding == "" {
		switch c.ServerMode() {
		case "http", "https", "whttp":
			if h := c.request.(whttp.Request).Header().Get(whttp.HeaderAcceptEncoding); h != "" {
				for _, v := range strings.Split(h, ";") {
					if strings.Contains(v, "gzip") { // we do Contains because sometimes browsers has the q=, we don't use it atm. || strings.Contains(v,"deflate"){
						c.encoding = "gzip"
					}
				}
			}
		case "rpc", "wrpc", "grpc":
		default:
		}
	}
	return c.encoding
}

// content-encoding
func (c *Context) ContentEncoding() string {
	switch c.ServerMode() {
	case "http", "https", "whttp":
		return c.Response().(whttp.Response).Header().Get(whttp.HeaderContentEncoding)
	case "rpc", "wrpc", "grpc":
	default:
	}
	return ""
}

func (c *Context) ContentType() string {
	switch c.ServerMode() {
	case "http", "https", "whttp":
		return c.Response().(whttp.Response).Header().Get(whttp.HeaderContentType)
	case "rpc", "wrpc", "grpc":
	default:
	}
	return ""
}

// userIP
func (c *Context) UserIP() string {
	switch c.ServerMode() {
	case "http", "https", "whttp":
		if uip := c.Request().(whttp.Request).Header().Get(whttp.HeaderXIp); uip != "" {
			// 由微服务透传过来
			return uip
		} else if xff := c.Request().(whttp.Request).Header().Get(whttp.HeaderXForwardedFor); xff != "" {
			i := strings.Index(xff, ", ")
			if i == -1 {
				i = len(xff)
			}
			return xff[:i]
		} else if xrip := c.Request().(whttp.Request).Header().Get(whttp.HeaderXRealIP); xrip != "" {
			return xrip
		}
		return c.ClientIP()
	case "rpc", "wrpc", "grpc":
		return c.ClientIP()
	default:
	}
	return ""
}

// userid
func (c *Context) UserID() string {
	switch c.ServerMode() {
	case "http", "https", "whttp":
		return c.Request().(whttp.Request).Header().Get(whttp.HeaderXUserId)
	case "rpc", "wrpc", "grpc":
	default:
	}
	return ""
}

// depth
func (c *Context) Depth() uint64 {
	switch c.ServerMode() {
	case "http", "https", "whttp":
		if ds := c.Request().(whttp.Request).Header().Get(whttp.HeaderXDepth); ds != "" {
			if depth, err := strconv.ParseUint(ds, 0, 64); err == nil {
				return depth + 1
			}
		}
	}
	return 0
}

// from
func (c *Context) From() string {
	switch c.ServerMode() {
	case "http", "https", "whttp":
		return c.Request().(whttp.Request).Header().Get(whttp.HeaderXAppId)
	}
	return ""
}

// get pre request id
func (c *Context) PreRequestId() string {
	switch c.ServerMode() {
	case "http", "https", "whttp":
		return c.Request().(whttp.Request).Header().Get(whttp.HeaderXRequestId)
	}
	return ""
}

// cfg
func (c *Context) Cfg() *environ.Config {
	return Cfg()
}

// router
func (c *Context) SetNode(node interface{}) {
	c.node = node
}

func (c *Context) Node() interface{} {
	return c.node
}

// no cache
// 这个配置由业务代码决定, 可以overwrite配置
func (c *Context) SetNoCache(b bool) {
	c.noCache = b
}
func (c *Context) NoCache() bool {
	return c.noCache
}

// headers
func (c *Context) AddHeader(key, value string) {
	switch c.ServerMode() {
	case "http", "https", "whttp":
		c.Response().(whttp.Response).Header().Add(key, value)
	case "rpc", "wrpc", "grpc":
		c.Response().(*wrpc.Response).Header().Add(key, value)
	}
}

func (c *Context) SetHeader(key, value string) {
	switch c.ServerMode() {
	case "http", "https", "whttp":
		c.Response().(whttp.Response).Header().Set(key, value)
	case "rpc", "wrpc", "grpc":
		c.Response().(*wrpc.Response).Header().Set(key, value)
	}
}

// flush
func (c *Context) Flush() {
	switch c.ServerMode() {
	case "http", "https", "whttp":
		c.Response().(whttp.Response).Flush()
	case "rpc", "wrpc", "grpc":
		c.Response().(*wrpc.Response).Flush(c.request.(*wrpc.Request).Context())
	}
}
