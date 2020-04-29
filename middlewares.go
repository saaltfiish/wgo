package wgo

// 这里定义wgo级别的middleware, 同时各子模块也可以有自己的middleware, 并且可以方便的进行转换

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"

	"wgo/utils"
	"wgo/whttp"
	// "github.com/dustin/randbo"
)

// default middlewares
type (
	// HandlerFunc defines an interface to serve wgo requests
	HandlerFunc func(*Context) error

	MiddlewareFunc func(HandlerFunc) HandlerFunc

	// RecoverConfig defines the config for recover middleware.
	RecoverConfig struct {
		// Size of the stack to be printed.
		// Optional. Default value 4KB.
		StackSize int `json:"stack_size"`

		// DisableStackAll disables formatting stack traces of all other goroutines
		// into buffer after the trace for the current goroutine.
		// Optional. Default value false.
		DisableStackAll bool `json:"disable_stack_all"`

		// DisablePrintStack disables printing stack trace.
		// Optional. Default value as false.
		DisablePrintStack bool `json:"disable_print_stack"`
	}
)

func (h MiddlewareFunc) Name() string {
	t := reflect.ValueOf(h).Type()
	if t.Kind() == reflect.Func {
		return runtime.FuncForPC(reflect.ValueOf(h).Pointer()).Name()
	}
	return t.String()
}

var (
	// DefaultRecoverConfig is the default recover middleware config.
	DefaultRecoverConfig = RecoverConfig{
		StackSize:         4 << 10, // 4 KB
		DisableStackAll:   false,
		DisablePrintStack: false,
	}
)

// Recover returns a middleware which recovers from panics anywhere in the chain
// and handles the control to the centralized HTTPErrorHandler.
func Recover() MiddlewareFunc {
	return RecoverWithConfig(DefaultRecoverConfig)
}

// RecoverWithConfig returns a recover middleware from config.
func RecoverWithConfig(config RecoverConfig) MiddlewareFunc {
	// Defaults
	if config.StackSize == 0 {
		config.StackSize = DefaultRecoverConfig.StackSize
	}

	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			c.Debug("[wgo.Recover]-->%s<--", c.Query())
			defer func() {
				if r := recover(); r != nil {
					var err error
					switch r := r.(type) {
					case error:
						err = r
					default:
						err = fmt.Errorf("%v", r)
					}
					stack := make([]byte, config.StackSize)
					length := runtime.Stack(stack, !config.DisableStackAll)
					if !config.DisablePrintStack {
						c.Error("[%s %s", err, stack[:length])
					}
					c.ERROR(err)
				}
			}()
			return next(c)
		}
	}
}

// 准备工作
// 判断特别参数
// 生成request_id
func Prepare() MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) (err error) {
			c.Debug("[wgo.Prepare]-->%s<--", c.Query())
			if c.Request().(whttp.Request).Method() == "OPTIONS" { // 统一处理options请求
				c.response.(whttp.Response).Header().Set(whttp.HeaderAccessControlMaxAge, "86400")
				c.response.(whttp.Response).Header().Set(whttp.HeaderAccessControlAllowMethods, "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				if ch := c.Request().(whttp.Request).Header().Get(whttp.HeaderAccessControlRequestHeaders); ch != "" {
					c.response.(whttp.Response).Header().Set(whttp.HeaderAccessControlAllowHeaders, ch) // 来者不拒
				}
				if origin := c.Request().(whttp.Request).Header().Get(whttp.HeaderOrigin); origin != "" {
					c.response.(whttp.Response).Header().Set(whttp.HeaderAccessControlAllowOrigin, origin)
					c.response.(whttp.Response).Header().Set(whttp.HeaderAccessControlAllowCredentials, "true")
				}
				c.Response().(whttp.Response).WriteHeader(whttp.StatusOK)
				return nil
			}
			// find request id
			requestId := ""
			if prid := c.PreRequestId(); prid != "" && c.Depth() > 0 {
				requestId = prid
			} else { // generate request id
				requestId = utils.FastRequestId(16)
				// not behind msa,
				// cors header(Access-Control-Allow-*)
				if origin := c.Request().(whttp.Request).Header().Get(whttp.HeaderOrigin); origin != "" {
					c.response.(whttp.Response).Header().Set(whttp.HeaderAccessControlAllowOrigin, origin)
					c.response.(whttp.Response).Header().Set(whttp.HeaderAccessControlAllowCredentials, "true")
				}
			}
			c.SetRequestID(requestId)

			err = next(c)

			if c.ServerMode() == "http" {
				c.response.(whttp.Response).Header().
					Set(whttp.HeaderServer, fmt.Sprintf("%s %s", strings.ToUpper(Env().ProcName), Version()))
			}

			return err
		}
	}
}
