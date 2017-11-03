package server

import ()

type (
	ContextGenFunc func(req interface{}, res interface{}) interface{}
	//Context interface {
	//	Context() ctx.Context
	//	SetContext(ctx.Context)
	//	Deadline() (time.Time, bool)
	//	Done() <-chan struct{}
	//	Err() error
	//	Value(key interface{}) interface{}
	//	// request & response
	//	Request() interface{}
	//	Response() interface{}
	//	// logger
	//	SetLogger(interface{})
	//	Logger() Logger
	//	// server mode (http/rpc/https)
	//	ServerMode() string
	//	// params
	//	Param(string) string
	//	ParamNames() []string
	//	SetParamNames(...string)
	//	ParamValues() []string
	//	SetParamValues(...string)
	//	// return error
	//	ERROR(error)
	//	// request id
	//	SetRequestID(string)
	//	RequestID() string
	//	// Ext return ext content
	//	SetExt(interface{})
	//	Ext() interface{}
	//	// mux
	//	SetMux(Mux)
	//	Mux() Mux
	//}
)
