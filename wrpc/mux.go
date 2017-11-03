package wrpc

import (
	"sync"

	"wgo/server"
	"wgo/utils"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

type (
	// Mux is the top-level framework instance.
	Mux struct {
		cgen       func() interface{} // context generator
		mconv      func(...interface{}) []*Middleware
		middleware []*Middleware
		sd         *grpc.ServiceDesc
		router     map[string]HandlerFunc
		logger     server.Logger
		pool       sync.Pool // context pool
		engine     server.Engine
	}
)

func NewMux(gen func() interface{}, conv func(...interface{}) []*Middleware) *Mux {
	m := &Mux{
		cgen:   gen,  // context 创建
		mconv:  conv, // middleware 转换
		router: make(map[string]HandlerFunc),
	}

	m.pool = sync.Pool{
		New: func() interface{} {
			return m.NewContext((*Request)(nil), (*Response)(nil))
		},
	}
	return m
}

// NewContext returns a Context instance.
func (m *Mux) NewContext(req *Request, res *Response) Context {
	c := m.cgen().(Context)
	c.SetLogger(m.logger)
	c.SetMux(m)
	return c
}

// Middleware
func (m *Mux) Middlewares(ms ...interface{}) []*Middleware {
	return m.mconv(ms...)
}

func (m *Mux) Prepare() {
}

// Use adds middleware to the chain which is run after router.
func (m *Mux) Use(ms ...interface{}) {
	m.middleware = append(m.middleware, m.Middlewares(ms...)...)
}

// engine
func (m *Mux) SetEngine(e server.Engine) {
	m.engine = e
}
func (m *Mux) Engine() server.Engine {
	return m.engine
}

// Logger returns the logger instance.
func (m *Mux) Logger() server.Logger {
	return m.logger
}

// SetLogger defines a custom logger.
func (m *Mux) SetLogger(l interface{}) {
	m.logger = l.(server.Logger)
}

// 增加路由
func (m *Mux) Add(methodName string, h HandlerFunc) {
	// middleware chain
	if ml := len(m.middleware); ml > 0 {
		aum := make([]string, 0)
		ms := m.middleware
		for i := ml - 1; i >= 0; i-- {
			if !utils.InSliceIgnorecase(ms[i].tag, aum) {
				h = ms[i].Func(h)
				aum = append(aum, ms[i].tag)
			}
		}
	}
	m.router[methodName] = h
}

func (m *Mux) Serve(req interface{}, res interface{}) {
	c := m.pool.Get().(Context)
	defer m.pool.Put(c)
	c.RPCReset(req.(*Request), res.(*Response))

	// route by method name
	if h, ok := m.router[req.(*Request).Method()]; ok {
		if err := h(c); err != nil {
			// todo
			res.(*Response).Err = Err(err)
		}
	} else { // not found
		res.(*Response).Err = NewError(codes.NotFound, "not found route")
	}
}
