// Package rest provides ...
package rest

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"wgo"
	"wgo/utils"
	"wgo/whttp"
)

// rest pools, key是tablename
var restPool = utils.NewSafeMap()

type REST struct {
	Count int64   `json:"count,omitempty" db:"-" filter:",H,G,D"` // 计数
	Sum   float64 `json:"sum,omitempty" db:"-" filter:",H,G,D"`   // 求和

	name      string               `db:"-"`
	endpoint  string               `db:"-"`
	model     Model                `db:"-"`
	zoo       *utils.SafeMap       `db:"-"`
	columns   []utils.StructColumn `db:"-"` // 缓存，不需要每次都ReadStructColumns
	defaultms []interface{}        `db:"-"` // 默认的middlewares
	mg        func() Model         `db:"-"` // 生成新model的程序, model generator
	pool      *sync.Pool           `db:"-"`

	ctx         *wgo.Context                   `db:"-"`
	transaction *Transaction                   `db:"-"`
	keeper      func(utils.StructColumn) error `db:"-"`
	conditions  []*Condition                   `db:"-"`
	pagination  *Pagination                    `db:"-"`
	fields      []string                       `db:"-"`
	result      interface{}                    `db:"-"`
	newer       Model                          `db:"-"`
	older       Model                          `db:"-"`
	filled      bool                           `db:"-"` //是否有内容
	saved       bool                           `db:"-"` // 是否已存储
	guest       bool                           `db:"-"` // guest代表不是这个请求endpoint对应的Model
}

func init() {
	SetLogger(wgo.Self())
	// wgo.Use(Init())
	// wgo.Use(Auth())

	// try register self
	// behind SetLogger
	RegisterConfig(wgo.Env().ProcName)
}

// 新建一个REST的工厂, 闭包
func restFactory(endpoint string, i interface{}, ms ...interface{}) func() interface{} {
	mg := modelFactory(i)
	name := getTableName(i)
	if endpoint == "" {
		endpoint = utils.Pluralize(name)
	}
	return func() interface{} {
		rest := &REST{
			name:      name,
			endpoint:  endpoint,
			zoo:       zoo.Clone(),
			defaultms: append(defaultMiddlewares, ms...),
			pool:      &sync.Pool{New: restFactory(endpoint, i, ms...)},
			mg:        mg,
		}
		rest.setModel(rest.mg())
		return rest
	}
}

// add rest
func addREST(m Model, opts ...interface{}) *REST {
	endpoint := utils.NewParams(opts).StringByIndex(0)
	rest := restFactory(endpoint, m)().(*REST)
	Debug("[addREST]adding rest: %s, custom endpoint: %s", rest.Name(), endpoint)

	// 生成rest pool并存储, 运行时rest,model的创建都依赖这个pool
	restPool.Set(rest.Name(), rest.Pool())
	return rest
}

// 获取跟i相关的REST
// sence: 只知道i, 通过i的名字找到pool并生成新的*REST
func getREST(i interface{}) *REST {
	if m := modelFactory(i)(); m != nil {
		name := underscore(strings.TrimSuffix(reflect.Indirect(reflect.ValueOf(m)).Type().Name(), "Table"))
		if pool := restPool.Get(name); pool != nil {
			Debug("[getREST]get %s's rest from pool!", name)
			return pool.(*sync.Pool).Get().(*REST)
		}
	}
	return nil
}

// get/build rest instance
// sence: deal with request, with context
func GetREST(c *wgo.Context) *REST {
	if r := c.Get("__!rest!__"); r != nil {
		if rest, ok := r.(*REST); ok {
			return rest
		}
	}
	if pi := c.Options(optionKey(ModelPoolKey)); pi != nil {
		// Debug("[GetREST]get rest from pool!!")
		// get from pool
		rest := pi.(*sync.Pool).Get().(*REST)

		// inject context
		rest.setContext(c)
		c.Set("__!rest!__", rest)
		return rest
	} else {
		Warn("[GetREST]not found pool: %s", c.Query())
	}

	return nil
}

// rest *REST, before Put it back to pool
func (r *REST) reset() *REST {
	r.ctx = nil
	r.transaction = nil
	r.keeper = nil
	r.conditions = nil
	r.fields = nil
	r.newer = nil
	r.older = nil
	r.filled = false
	r.saved = false
	r.guest = false
	r.setModel(r.mg())

	return r
}

// release to pool
func (r *REST) release() {
	// reset model, avoid model cached
	// release的时候先reset再Put, 这样Get之后就不需要了
	r.reset()
	if r.Context() != nil {
		r.Context().Set("__!rest!__", nil)
	}
	if r.Pool() != nil {
		r.Pool().Put(r)
	}
}

// new rest from pool
// sence: create a *REST from old *REST's pool
func (r *REST) newREST() *REST {
	if r == nil {
		return nil
	}
	if pool := r.Pool(); pool != nil {
		rest := pool.Get().(*REST)
		if c := r.Context(); c != nil { // 尽量传递context
			rest.setContext(c)
		}
		return rest
	}
	return nil
}

// properties
func (r *REST) Name() string {
	if r == nil {
		return ""
	}
	return r.name
}
func (r *REST) Endpoint() string {
	if r == nil {
		return ""
	}
	// return utils.MustString(r.Options(EndpointKey))
	return r.endpoint
}
func (r *REST) Pool() *sync.Pool {
	if r == nil {
		return nil
	}
	return r.pool
}

// columns
func (r *REST) Columns() []utils.StructColumn {
	if r == nil {
		return nil
	}
	return r.columns
}

// zoo
func (r *REST) Zoo() *utils.SafeMap {
	if r == nil {
		return nil
	}
	return r.zoo
}

// context
func (r *REST) Context() *wgo.Context {
	if r == nil {
		return nil
	}
	return r.ctx
}
func (r *REST) setContext(c *wgo.Context) *REST {
	if r == nil {
		return nil
	}
	r.ctx = c
	return r
}
func (r *REST) setGuest() {
	r.guest = true
}
func (r *REST) isGuest() bool {
	return r.guest
}

// result
func (rest *REST) SetResult(rt interface{}) (interface{}, error) {
	if rest == nil {
		return nil, errors.New("REST is nil")
	}
	rest.result = rt
	return rest.result, nil
}
func (rest *REST) Result() interface{} {
	if rest == nil {
		return nil
	}
	return rest.result
}

// values
func (rest *REST) Set(key string, val interface{}) {
	rest.ctx.Set(key, val)
}

func (rest *REST) Get(key string) interface{} {
	return rest.ctx.Get(key)
}

// env
func (rest *REST) SetEnv(k string, v interface{}) {
	if k != "" {
		// rest.env[k] = v
		ek := fmt.Sprintf("%s:%s", RESTKey, k)
		rest.ctx.Set(ek, v)
	}
}

func (rest *REST) GetEnv(k string) interface{} {
	ek := fmt.Sprintf("%s:%s", RESTKey, k)
	return rest.ctx.Get(ek)
}

// action
// func (rest *REST) SetAction(act string) {
// 	rest.SetEnv("_action_", act)
// }
// func (rest *REST) Action() string {
// 	acti := rest.GetEnv("_action_")
// 	if act, ok := acti.(string); ok {
// 		return act
// 	}
// 	return ""
// }

// creating
func (rest *REST) Creating() bool {
	if m := rest.Model(); m != nil {
		cc := CanCreate(m)
		if !cc {
			return cc
		}
		if ctx := rest.Context(); ctx != nil && !rest.isGuest() {
			// 主REST, 判断method
			switch m := ctx.Method(); m {
			case "POST", "PUT":
				return cc
			default:
				return false
			}
		}
		return cc
	}
	return false
}

// updating
func (rest *REST) Updating() bool {
	if m := rest.Model(); m != nil {
		cu := CanUpdate(m)
		if !cu {
			return cu
		}
		if ctx := rest.Context(); ctx != nil && !rest.isGuest() {
			// 主REST, 判断method
			switch m := ctx.Method(); m {
			case "PATCH", "POST", "PUT":
				return cu
			default:
				return false
			}
		}
		return cu
	}
	return false
}

// response
// ok
func (rest *REST) OK(data interface{}) (err error) {
	c := rest.Context()
	return c.JSON(getCode(true, c.Request().(whttp.Request).Method()), data)
}

// not ok
func (rest *REST) NotOK(m interface{}) (err error) {
	c := rest.Context()
	code := getCode(false, c.Request().(whttp.Request).Method())
	code *= 1000
	msg := "have errors!"
	if _, ok := m.(error); ok {
		msg = m.(error).Error()
	} else if _, ok := m.(string); ok {
		msg = m.(string)
	}
	return c.NewError(code, msg)
}

// bad request
func (rest *REST) BadRequest(m interface{}) (err error) {
	c := rest.Context()
	code := whttp.StatusBadRequest * 1000
	msg := "bad request"
	if _, ok := m.(error); ok {
		msg = m.(error).Error()
	} else if _, ok := m.(string); ok {
		msg = m.(string)
	}
	return c.NewError(code, msg)
}

// not found
func (rest *REST) NotFound(m interface{}) (err error) {
	c := rest.Context()
	code := whttp.StatusNotFound * 1000
	msg := "not found"
	if _, ok := m.(error); ok {
		msg = m.(error).Error()
	} else if _, ok := m.(string); ok {
		msg = m.(string)
	}
	return c.NewError(code, msg)
}

// internal error
func (rest *REST) InternalError(m interface{}) (err error) {
	code := whttp.StatusInternalServerError * 1000
	msg := "internal errors!"
	if _, ok := m.(error); ok {
		msg = m.(error).Error()
	} else if _, ok := m.(string); ok {
		msg = m.(string)
	}
	return rest.Context().NewError(code, msg)
}

// 获取返回码
func getCode(ifSuc bool, m string) (s int) {
	method := strings.ToLower(m)
	if ifSuc {
		switch method {
		case "get":
			return whttp.StatusOK
		case "delete":
			return whttp.StatusNoContent
		case "put":
			return whttp.StatusCreated
		case "post":
			return whttp.StatusCreated
		case "patch":
			// return whttp.StatusResetContent
			return whttp.StatusOK // 大多数浏览器看到205就不显示返回内容了
		case "head":
			return whttp.StatusOK
		default:
			return whttp.StatusOK
		}
	} else {
		switch method {
		case "get":
			return whttp.StatusNotFound
		case "delete":
			//return whttp.StatusNotAcceptable
			return whttp.StatusBadRequest
		case "put":
			//return whttp.StatusNotAcceptable
			return whttp.StatusBadRequest
		case "post":
			//return whttp.StatusNotAcceptable
			return whttp.StatusBadRequest
		case "patch":
			//return whttp.StatusNotAcceptable
			return whttp.StatusBadRequest
		case "head":
			return whttp.StatusConflict
		default:
			return whttp.StatusBadRequest
		}
	}
}
