// Package rest provides ...
package rest

import (
	"reflect"
	"strings"

	"wgo"
	"wgo/utils"
	"wgo/whttp"
)

type REST struct {
	Count      int64                       `json:"count,omitempty" db:"-" filter:",H,G,D"` // 计数
	Sum        float64                     `json:"sum,omitempty" db:"-" filter:",H,G,D"`   // 求和
	endpoint   string                      `db:"-"`
	model      Model                       `db:"-"`
	action     string                      `db:"-"`
	ctx        *wgo.Context                `db:"-"`
	env        map[interface{}]interface{} `db:"-"`
	keeper     Keeper                      `db:"-"`
	conditions []*Condition                `db:"-"`
	pagination *Pagination                 `db:"-"`
	fields     []string                    `db:"-"`
	older      Model                       `db:"-"`
	filled     bool                        `db:"-"` //是否有内容
	defaultms  []interface{}               `db:"-"` // 默认的middlewares
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
func NewREST(c *wgo.Context) (rest *REST) {
	rest = new(REST)
	rest.ctx = c
	rest.env = make(map[interface{}]interface{})
	c.SetExt(rest)
	return
}

// SetModel
func (rest *REST) SetModel(m Model) Model {
	rest.model = m
	// 注入m
	rest.importTo(m)
	return m
}

// 把rest注入i
func (rest *REST) importTo(i interface{}, fields ...string) {
	field := "REST"
	if len(fields) > 0 {
		field = fields[0]
	}
	if fv := utils.FieldByName(i, field); fv.IsValid() {
		if fv.Kind() == reflect.Ptr {
			fv.Set(reflect.ValueOf(rest))
		} else {
			fv.Set(reflect.ValueOf(rest).Elem())
		}
	}
}

// Model
func (rest *REST) Model() Model {
	return rest.model
}

// release
func (rest *REST) Release() {
	if rest.ctx != nil {
		rest.ctx.SetExt(nil)
	}
}

// context
func (rest *REST) Context() *wgo.Context {
	return rest.ctx
}
func (rest *REST) SetContext(c *wgo.Context) {
	rest.ctx = c
}

// env
func (rest *REST) SetEnv(k string, v interface{}) {
	if k != "" {
		rest.env[k] = v
	}
}

func (rest *REST) GetEnv(k string) interface{} {
	if v, ok := rest.env[k]; ok {
		return v
	}
	return nil
}

// action
func (rest *REST) SetAction(act string) {
	rest.action = act
}
func (rest *REST) Action() string {
	return rest.action
}

// creating
func (rest *REST) Creating() bool {
	return rest.action == "creating"
}

// updating
func (rest *REST) Updating() bool {
	return rest.action == "updating"
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
			return whttp.StatusResetContent
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
