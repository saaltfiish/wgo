package fasthttp

import (
	"net"
	"sync"
	"time"

	"wgo/server"

	"github.com/valyala/fasthttp"
)

type (
	// Server implements `whttp.Handler`.
	Engine struct {
		*fasthttp.Server
		mux  server.Mux
		pool *pool
		name string
	}

	pool struct {
		request        sync.Pool
		response       sync.Pool
		requestHeader  sync.Pool
		responseHeader sync.Pool
		url            sync.Pool
	}
)

// New returns `Engine`
func New() (eng *Engine) {
	eng = &Engine{
		//Server: new(fasthttp.Server),
		Server: &fasthttp.Server{
			Name:               "WGO",
			ReduceMemoryUsage:  true,
			Concurrency:        100000,
			ReadTimeout:        180 * time.Second,
			WriteTimeout:       90 * time.Second,
			MaxRequestBodySize: 64 * 1024 * 1024,
			ReadBufferSize:     16 * 1024,
			LogAllErrors:       true,
		},
		pool: &pool{
			request: sync.Pool{
				New: func() interface{} {
					return &Request{}
				},
			},
			response: sync.Pool{
				New: func() interface{} {
					return &Response{}
				},
			},
			requestHeader: sync.Pool{
				New: func() interface{} {
					return &RequestHeader{}
				},
			},
			responseHeader: sync.Pool{
				New: func() interface{} {
					return &ResponseHeader{}
				},
			},
			url: sync.Pool{
				New: func() interface{} {
					return &URL{}
				},
			},
		},
		name: "fasthttp",
	}
	eng.Handler = eng.ServeHTTP
	return
}

// Mux
func (e *Engine) SetMux(m server.Mux) {
	e.mux = m
}
func (e *Engine) Mux() server.Mux {
	return e.mux
}

// name
func (e *Engine) Name() string {
	return e.name
}

// Serve
func (e *Engine) Start(l net.Listener) error {
	return e.Server.Serve(l)
}

// handler
//func (e *Engine) SetHandler(h func(interface{}, interface{})) {
//	e.handler = h
//}

func (e *Engine) ServeHTTP(c *fasthttp.RequestCtx) {
	// Request
	req := e.pool.request.Get().(*Request)
	reqHdr := e.pool.requestHeader.Get().(*RequestHeader)
	reqURL := e.pool.url.Get().(*URL)
	reqHdr.reset(&c.Request.Header)
	reqURL.reset(c.URI())
	req.reset(c, reqHdr, reqURL)

	// Response
	res := e.pool.response.Get().(*Response)
	resHdr := e.pool.responseHeader.Get().(*ResponseHeader)
	resHdr.reset(&c.Response.Header)
	res.reset(c, resHdr)

	e.mux.Serve(req, res)

	// Return to pool
	e.pool.request.Put(req)
	e.pool.requestHeader.Put(reqHdr)
	e.pool.url.Put(reqURL)
	e.pool.response.Put(res)
	e.pool.responseHeader.Put(resHdr)
}
