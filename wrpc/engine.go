package wrpc

import (
	"context"
	"net"
	"strings"
	"sync"

	"wgo/server"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
)

type (
	RegisterFunc func(*grpc.Server)

	Engine struct {
		*grpc.Server
		mux  server.Mux
		name string
	}
)

// wrpc newEngine
func newEngine() server.Engine {
	// interceptor
	e := &Engine{name: "grpc"}
	e.Server = grpc.NewServer(grpc.UnaryInterceptor(e.InterceptorWrapper()))
	return e
}

// engine Vendor
// 把能生成for wrpc的 server.Engine 放到server.Server
func Factory(s *server.Server, cgen func() interface{}, mconv func(...interface{}) []*Middleware) *server.Server {
	// engine factory func
	var ef server.EngineFactory
	ef = func() server.Engine {
		return newEngine()
	}
	// mux factory func
	var mf server.MuxFactory
	mf = func() server.Mux {
		return NewMux(s.EngineName(), cgen, mconv)
	}
	return s.Factory(ef, mf)
}

/* {{{ func (e *Engine) Mux()
 *
 */
func (e *Engine) Mux() server.Mux {
	return e.mux
}

/* }}} */

// name
func (e *Engine) Name() string {
	return e.name
}

/* {{{ func (e *Engine) SetMux(m server.Mux)
 *
 */
func (e *Engine) SetMux(m server.Mux) {
	e.mux = m
}

/* }}} */

/* {{{ func (e *Engine) Start(l net.Listener) error
 *
 */
func (e *Engine) Start(l net.Listener) error {
	// 重置方法
	//sd := e.Server.ServiceDesc()
	//if len(sd.Methods) > 0 {
	//	for i := range sd.Methods {
	//		sd.Methods[i].Handler = e.HandlerWrapper(sd.Methods[i].MethodName)
	//	}
	//} else {
	//	e.Mux().(*Mux).Logger().Info("not found ServiceDesc")
	//}
	return e.Server.Serve(l)
}

/* }}} */

/* {{{ func RegisterService(rf RegisterFunc)
 * 注册服务
 */
func (e *Engine) RegisterService(rf RegisterFunc) {
	reflection.Register(e.Server)
	e.Mux().Logger().Info("RegisterService")
	rf(e.Server)
}

/* }}} */

func (e *Engine) InterceptorWrapper() func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	reqPool := sync.Pool{
		New: func() interface{} {
			return &Request{}
		},
	}
	resPool := sync.Pool{
		New: func() interface{} {
			return &Response{}
		},
	}

	return func(ctx context.Context, request interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (respose interface{}, err error) {
		// request
		req := reqPool.Get().(*Request)
		defer reqPool.Put(req)
		req.Body = request
		req.context = ctx
		// request headers
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			req.header = &Header{MD: md}
		}
		// find method
		fs := strings.Split(info.FullMethod, "/")
		method := fs[len(fs)-1]
		//Info("full: %s, rpc method: %s", info.FullMethod, method)
		req.method = method

		// response
		res := resPool.Get().(*Response)
		defer resPool.Put(res)
		res.Body = nil
		res.Err = nil
		res.header = &Header{MD: metadata.Pairs()}

		e.mux.Serve(req, res)

		return res.Body, res.Err
	}
}
