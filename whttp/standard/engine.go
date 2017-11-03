package standard

import (
	"bytes"
	"net"
	"net/http"
	"sync"

	"wgo/server"
)

type (
	// Server implements `server.Handler`.
	Engine struct {
		*http.Server
		mux  server.Mux
		pool *pool
		name string
	}

	pool struct {
		request         sync.Pool
		response        sync.Pool
		responseAdapter sync.Pool
		header          sync.Pool
		url             sync.Pool
	}
)

// New returns `Server` instance with provided listen address.
func New() (eng *Engine) {
	eng = &Engine{
		Server: new(http.Server),
		pool: &pool{
			request: sync.Pool{
				New: func() interface{} {
					return &Request{}
				},
			},
			response: sync.Pool{
				New: func() interface{} {
					return &Response{
						buffer: bytes.NewBuffer([]byte{}),
					}
				},
			},
			responseAdapter: sync.Pool{
				New: func() interface{} {
					return &responseAdapter{}
				},
			},
			header: sync.Pool{
				New: func() interface{} {
					return &Header{}
				},
			},
			url: sync.Pool{
				New: func() interface{} {
					return &URL{}
				},
			},
		},
		name: "standard",
	}
	eng.Handler = eng
	return eng
}

// Mux
func (e *Engine) SetMux(m server.Mux) server.Mux {
	e.mux = m
	return e.mux
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

// ServeHTTP implements `http.Handler` interface.
func (e *Engine) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Request
	req := e.pool.request.Get().(*Request)
	reqHdr := e.pool.header.Get().(*Header)
	reqURL := e.pool.url.Get().(*URL)
	reqHdr.reset(r.Header)
	reqURL.reset(r.URL)
	req.reset(r, reqHdr, reqURL)

	// Response
	//w.Header().Set("Server", "WGO")
	res := e.pool.response.Get().(*Response)
	resAdpt := e.pool.responseAdapter.Get().(*responseAdapter)
	resAdpt.reset(res)
	resHdr := e.pool.header.Get().(*Header)
	resHdr.reset(w.Header())
	res.reset(w, resAdpt, resHdr)

	e.mux.Serve(req, res)

	// Return to pool
	e.pool.request.Put(req)
	e.pool.header.Put(reqHdr)
	e.pool.url.Put(reqURL)
	e.pool.response.Put(res)
	e.pool.header.Put(resHdr)
}
