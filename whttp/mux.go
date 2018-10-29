package whttp

import (
	"strings"
	"sync"

	"wgo/server"
	"wgo/whttp/fasthttp"
	"wgo/whttp/standard"
)

type (
	// Mux is the top-level framework instance.
	Mux struct {
		name            string
		cgen            func() interface{} // context generator
		mconv           func(...interface{}) []*Middleware
		middleware      []*Middleware
		notFoundHandler HandlerFunc
		binder          Binder
		router          *Router
		logger          server.Logger
		pool            sync.Pool // context pool
		// engine          server.Engine
	}

	// Validator is the interface that wraps the Validate fundnction.
	Validator interface {
		Validate() error
	}
)

var (
	Methods = [...]string{
		METHOD_CONNECT,
		METHOD_DELETE,
		METHOD_GET,
		METHOD_HEAD,
		METHOD_OPTIONS,
		METHOD_PATCH,
		METHOD_POST,
		METHOD_PUT,
		METHOD_TRACE,
	}
)

// Error handlers
var (
	NotFoundHandler = func(c Context) error {
		return ErrNotFound
	}

	MethodNotAllowedHandler = func(c Context) error {
		return ErrMethodNotAllowed
	}
)

func NewMux(name string, gen func() interface{}, conv func(...interface{}) []*Middleware) *Mux {
	// Debug("[NewMux]name: %s", name)
	m := &Mux{
		name:            name,
		cgen:            gen,  // context 创建
		mconv:           conv, // middleware 转换
		notFoundHandler: NotFoundHandler,
	}
	m.router = NewRouter(m) // router内部需要保留一份mux的指针, 所以传进去

	m.pool = sync.Pool{
		New: func() interface{} {
			return m.NewContext(nil, nil)
		},
	}
	return m
}

// NewContext returns a Context instance.
func (m *Mux) NewContext(req Request, res Response) Context {
	c := m.cgen().(Context)
	pvalues := make([]string, m.router.Depth())
	c.SetParamValues(pvalues...)
	// c.SetLogger(m.logger)
	c.SetMux(m)
	return c
}

// NewResponse return a Response instance.
func (m *Mux) NewResponse() Response {
	switch m.Name() {
	case "standard":
		return standard.NewResponse()
	default:
		return fasthttp.NewResponse()
	}
}

// Middleware
func (m *Mux) Middlewares(ms ...interface{}) []*Middleware {
	return m.mconv(ms...)
}

func (m *Mux) Prepare() {
	m.Router().BuildRoutes()
}

// Router returns router.
func (m *Mux) Router() *Router {
	return m.router
}

// SetLogger defines a custom logger.
func (m *Mux) SetLogger(l interface{}) {
	m.logger = l.(server.Logger)
}

// Logger
func (m *Mux) Logger() server.Logger {
	return m.logger
}

func (m *Mux) SetBinder(b Binder) {
	m.binder = b
}

func (m *Mux) Binder() Binder {
	return m.binder
}

// Use adds middleware to the chain which is run after router.
func (m *Mux) Use(ms ...interface{}) {
	m.middleware = append(m.middleware, m.Middlewares(ms...)...)
}

// NotFound adds customize notfound handler
func (m *Mux) NotFound(h HandlerFunc) {
	m.notFoundHandler = h
}

// abandon middleware from middleware chain
func (m *Mux) Abandon(ms ...interface{}) *Mux {
	var mws = make([]*Middleware, 0)
	for _, m1 := range m.middleware {
		abandon := false
		for _, m2 := range ms {
			//if reflect.ValueOf(m1).Pointer() == reflect.ValueOf(m2).Pointer() {
			if m1.tag == m2.(*Middleware).tag {
				//Info("abandon mux middleware: %#+q, %#+q", m1.tag, m2.tag)
				abandon = true
				break
			}
		}
		if !abandon {
			mws = append(mws, m1)
		}
	}

	m.middleware = mws
	return m
}

func (m *Mux) Add(method, path string, handler HandlerFunc, ms ...interface{}) *Route {
	wms := []*Middleware{}
	if len(m.middleware) > 0 {
		wms = append(wms, m.middleware...)
		wms = append(wms, m.Middlewares(ms...)...)
	}
	r := &Route{
		Method:     method,
		Path:       path,
		Handler:    handler,
		Middleware: wms,
	}
	m.router.AddRoute(r)
	return r
}

// name
func (m *Mux) Name() string {
	return m.name
}

// engine
// func (m *Mux) SetEngine(e server.Engine) {
// 	m.engine = e
// }
// func (m *Mux) Engine() server.Engine {
// 	return m.engine
// }

// Routes returns the registered routes.
func (m *Mux) routes() Routes {
	return m.router.Routes()
}

func (m *Mux) Serve(req interface{}, res interface{}) {
	c := m.pool.Get().(Context)
	defer m.pool.Put(c)
	c.HTTPReset(req.(Request), res.(Response))

	// default not found
	h := m.notFoundHandler

	// find route
	// reset method
	if req.(Request).Method() == "POST" {
		switch pm := strings.ToUpper(req.(Request).URL().QueryParam(PARAM_METHOD)); pm {
		case "PATCH", "DELETE", "OPTIONS", "TRACE", "PUT": // 这几个method有的环境无法生成
			req.(Request).SetMethod(pm)
			c.Info("method chage to: %s", pm)
		default:
		}
	}
	if node := m.router.Find(req.(Request).Method(), req.(Request).URL().Path(), c.ParamValues()); node != nil {
		c.SetNode(node)
		c.SetPath(node.Path())
		c.SetParamNames(node.Names()...)

		if f, ok := node.Func.(func(Context) error); ok {
			h = HandlerFunc(f)
		}
	}

	if err := h(c); err != nil {
		m.Logger().Error("serve error: %s", err)
		c.ERROR(err)
	}
	res.(Response).Commit()
}
