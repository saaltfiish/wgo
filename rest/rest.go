// Package rest provides ...
package rest

import (
	"fmt"
	"strings"

	"wgo"
	"wgo/whttp"
)

type REST struct {
	Count int64   `json:"count,omitempty" db:"-" filter:",H,G,D"` // 计数
	Sum   float64 `json:"sum,omitempty" db:"-" filter:",H,G,D"`   // 求和

	endpoint    string                            `db:"-"`
	model       Model                             `db:"-"`
	transaction *Transaction                      `db:"-"`
	ctx         *wgo.Context                      `db:"-"`
	keeper      func(string) (interface{}, error) `db:"-"`
	conditions  []*Condition                      `db:"-"`
	pagination  *Pagination                       `db:"-"`
	fields      []string                          `db:"-"`
	new         interface{}                       `db:"-"`
	older       Model                             `db:"-"`
	filled      bool                              `db:"-"` //是否有内容
	saved       bool                              `db:"-"` // 是否已存储
	defaultms   []interface{}                     `db:"-"` // 默认的middlewares
	// env        map[interface{}]interface{} `db:"-"`
}

func init() {
	SetLogger(wgo.Self())
	wgo.Use(Init())
	wgo.Use(Auth())

	// try register self
	// behind SetLogger
	RegisterConfig(wgo.Env().ProcName)
}

// new rest
func NewREST(c *wgo.Context) *REST {
	r := new(REST).setContext(c)
	if base := r.Options(BaseModelKey); base != nil {
		r.NewModel(base)
	}
	c.Set("__!rest!__", r)
	return r
}

// get/build rest instance
func GetREST(c *wgo.Context) *REST {
	if r := c.Get("__!rest!__"); r != nil {
		if rest, ok := r.(*REST); ok {
			return rest
		}
	}
	return NewREST(c)
}

// release
func (r *REST) Release() {
	if r.Context() != nil {
		r.Context().Set("__!rest!__", nil)
	}
}

// context
func (r *REST) Context() *wgo.Context {
	return r.ctx
}
func (r *REST) setContext(c *wgo.Context) *REST {
	r.ctx = c
	return r
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
func (rest *REST) SetAction(act string) {
	rest.SetEnv("_action_", act)
}
func (rest *REST) Action() string {
	acti := rest.GetEnv("_action_")
	if act, ok := acti.(string); ok {
		return act
	}
	return ""
}

// creating
func (rest *REST) Creating() bool {
	return rest.Action() == ACTION_CREATE
}

// updating
func (rest *REST) Updating() bool {
	return rest.Action() == ACTION_UPDATE
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
