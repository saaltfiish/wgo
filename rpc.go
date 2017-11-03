// Package wgo provides ...
package wgo

import (
	"wgo/wrpc"

	"google.golang.org/grpc"
)

/* {{{ func mixWrpcMiddlewares(ms ...interface{}) []*wrpc.Middlewares
 * 把MiddlewareFunc or wrpc.MiddlewareFunc 转换为 wrpc.MiddlewareFunc
 */
func mixWrpcMiddlewares(ms ...interface{}) []*wrpc.Middleware {
	// 支持wrpc.Middleware, Middleware两种中间件
	wms := []*wrpc.Middleware{}
	if len(ms) > 0 {
		for _, m := range ms {
			if _, ok := m.(MiddlewareFunc); ok {
				wms = append(wms, newWrpcMiddleware(m.(MiddlewareFunc)))
			} else if _, ok := m.(wrpc.MiddlewareFunc); ok {
				wms = append(wms, wrpc.NewMiddleware(m.(wrpc.MiddlewareFunc).Name(), m.(wrpc.MiddlewareFunc)))
			}
		}
	}
	return wms
}

/* }}} */

// HandlerFunc to wrpc.HandlerFunc
func handlerFuncToWrpcHandlerFunc(h HandlerFunc) wrpc.HandlerFunc {
	return func(c wrpc.Context) error {
		return h(c.(*Context))
	}
}

// wrpc.HandlerFunc to HandlerFunc
func wrpcHandlerFuncToHandlerFunc(h wrpc.HandlerFunc) HandlerFunc {
	return func(c *Context) error {
		return h(c)
	}
}

// Middleware to wrpc.Middleware
func middlewareToWrpcMiddleware(m MiddlewareFunc) wrpc.MiddlewareFunc {
	return func(h wrpc.HandlerFunc) wrpc.HandlerFunc {
		return handlerFuncToWrpcHandlerFunc(m(wrpcHandlerFuncToHandlerFunc(h)))
	}
}

// new wrpc middleware
func newWrpcMiddleware(m MiddlewareFunc) *wrpc.Middleware {
	return wrpc.NewMiddleware(m.Name(), middlewareToWrpcMiddleware(m))
}

// wrpc.Middleware to Middleware
func wrpcMiddlewareToMiddleware(m wrpc.MiddlewareFunc) MiddlewareFunc {
	return func(h HandlerFunc) HandlerFunc {
		return wrpcHandlerFuncToHandlerFunc(m(handlerFuncToWrpcHandlerFunc(h)))
	}
}

// 注册RPC服务
func RegisterRPCService(rf func(s *grpc.Server)) {
	if ss := wgo.RPCServers(); len(ss) > 0 {
		ss.RegisterRPCService(rf)
	}
}
func (ss Servers) RegisterRPCService(rf func(s *grpc.Server)) {
	for _, s := range ss {
		if s.Mux() != nil {
			Warn("%s RegisterRPCService", s.Name())
			s.Engine().(*wrpc.Engine).RegisterService(rf)
		}
	}
}

// add rpc method
func AddRPC(methodName string, h HandlerFunc) {
	if ss := wgo.RPCServers(); len(ss) > 0 {
		ss.AddRPC(methodName, h)
	}
}
func (ss Servers) AddRPC(methodName string, h HandlerFunc) {
	for _, s := range ss {
		if s.Mux() != nil {
			Warn("add rpc, mehtod: %s", methodName)
			s.Mux().(*wrpc.Mux).Add(methodName, handlerFuncToWrpcHandlerFunc(h))
		}
	}
}
