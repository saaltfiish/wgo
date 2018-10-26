package wrpc

import (
	"context"
	"time"

	"wgo/environ"
	"wgo/server"
)

type (
	// wrpc context
	Context interface {
		// Context returns `net/context.Context`.
		Context() context.Context

		// SetContext sets `net/context.Context`.
		SetContext(context.Context)

		// Deadline returns the time when work done on behalf of this context
		// should be canceled.  Deadline returns ok==false when no deadline is
		// set.  Successive calls to Deadline return the same results.
		Deadline() (deadline time.Time, ok bool)

		// Done returns a channel that's closed when work done on behalf of this
		// context should be canceled.  Done may return nil if this context can
		// never be canceled.  Successive calls to Done return the same value.
		Done() <-chan struct{}

		// Err returns a non-nil error value after Done is closed.  Err returns
		// Canceled if the context was canceled or DeadlineExceeded if the
		// context's deadline passed.  No other values for Err are defined.
		// After Done is closed, successive calls to Err return the same value.
		Err() error

		// Value returns the value associated with this context for key, or nil
		// if no value is associated with key.  Successive calls to Value with
		// the same key returns the same result.
		Value(key interface{}) interface{}

		// Request returns `Request` interface.
		Request() interface{}

		// Request returns `Response` interface.
		Response() interface{}

		// mux
		SetMux(server.Mux)
		Mux() server.Mux

		// Logger returns the `Logger` instance.
		SetLogger(interface{})
		Logger() server.Logger

		// Start return start time of get context
		Start() time.Time
		// Sub return duration from start
		Sub() time.Duration

		// Param returns path parameter by name.
		Param(string) string

		// ParamNames returns path parameter names.
		ParamNames() []string

		// SetParamNames sets path parameter names.
		SetParamNames(...string)

		// ParamValues returns path parameter values.
		ParamValues() []string

		// SetParamValues sets path parameter values.
		SetParamValues(...string)

		// return error
		ERROR(error)
		// Reset resets the context after request completes. It must be called along
		// See `Mux#Serve()`
		RPCReset(*Request, *Response)

		// request id
		SetRequestID(string)
		RequestID() string

		// decode
		//Decode(interface{}) error

		// RPC response
		RPC(interface{})

		// useful mehtods
		ServerMode() string
		Cfg() *environ.Config
		Host() string
		Depth() uint64
		ClientIP() string

		//logging
		Debug(arg0 interface{}, args ...interface{})
		Info(arg0 interface{}, args ...interface{})
		Warn(arg0 interface{}, args ...interface{})
		Error(arg0 interface{}, args ...interface{})
	}

	// 生成context
	ContextGenFuc func(Request, Response) Context
)
