// Package rest provides ...
package rest

import (
	"fmt"
	"net/http"
	"strings"

	"wgo"
	"wgo/server"
	"wgo/whttp"
)

type Router interface {
	GET(*wgo.Context) error
	LIST(*wgo.Context) error
	POST(*wgo.Context) error
	PUT(*wgo.Context) error
	DELETE(*wgo.Context) error
	PATCH(*wgo.Context) error
	HEAD(*wgo.Context) error
	OPTIONS(*wgo.Context) error
	TRACE(*wgo.Context) error
}

type Options map[string]interface{}

type Routes struct {
	whttp.Routes
}

// GET
func (_ *REST) GET(c *wgo.Context) error {
	return server.NewError(http.StatusMethodNotAllowed, http.StatusText(http.StatusMethodNotAllowed))
}

// List
func (_ *REST) LIST(c *wgo.Context) error {
	return server.NewError(http.StatusMethodNotAllowed, http.StatusText(http.StatusMethodNotAllowed))
}

// POST
func (_ *REST) POST(c *wgo.Context) error {
	return server.NewError(http.StatusMethodNotAllowed, http.StatusText(http.StatusMethodNotAllowed))
}

//PUT
func (_ *REST) PUT(c *wgo.Context) error {
	return server.NewError(http.StatusMethodNotAllowed, http.StatusText(http.StatusMethodNotAllowed))
}

// DELETE
func (_ *REST) DELETE(c *wgo.Context) error {
	return server.NewError(http.StatusMethodNotAllowed, http.StatusText(http.StatusMethodNotAllowed))
}

// PATCH
func (_ *REST) PATCH(c *wgo.Context) error {
	return server.NewError(http.StatusMethodNotAllowed, http.StatusText(http.StatusMethodNotAllowed))
}

// HEAD
func (_ *REST) HEAD(c *wgo.Context) error {
	return server.NewError(http.StatusMethodNotAllowed, http.StatusText(http.StatusMethodNotAllowed))
}

// OPTIONS
func (_ *REST) OPTIONS(c *wgo.Context) error {
	return server.NewError(http.StatusMethodNotAllowed, http.StatusText(http.StatusMethodNotAllowed))
}

// TRACE
func (_ *REST) TRACE(c *wgo.Context) error {
	return server.NewError(http.StatusMethodNotAllowed, http.StatusText(http.StatusMethodNotAllowed))
}

// deny
func RESTDeny(c *wgo.Context) error {
	return server.NewError(http.StatusMethodNotAllowed, http.StatusText(http.StatusMethodNotAllowed))
}

// 注册路由
// 注册之后可以自动获得rest提供的通用方法,这是rest的核心价值之一
// 同时也可以自己写同名方法覆盖
func Register(endpoint string, i interface{}, flag int, ms ...interface{}) *REST {
	if _, ok := i.(Router); !ok {
		panic("input not Router")
	}
	// rt := i.(Router)
	if _, ok := i.(Model); !ok {
		panic("input not Model")
	}
	// 找到真实的m
	m := digModel(i.(Model))
	rest := new(REST)
	rest.endpoint = endpoint
	rest.model = m
	rest.defaultms = ms

	// default,deny
	// wgo.HEAD("/"+endpoint, RESTDeny)
	// wgo.GET("/"+endpoint+"/:"+RowkeyKey, RESTDeny)
	// wgo.GET("/"+endpoint, RESTDeny)
	// wgo.POST("/"+endpoint, RESTDeny)
	// wgo.DELETE("/"+endpoint+"/:"+RowkeyKey, RESTDeny)
	// wgo.PATCH("/"+endpoint+"/:"+RowkeyKey, RESTDeny)
	// wgo.PUT("/"+endpoint+"/:"+RowkeyKey, RESTDeny)

	rest.Builtin(flag).SetOptions(BaseModelKey, m).SetOptions(EndpointKey, endpoint)
	return rest
}

// 内置方法
func (r *REST) Builtin(flag int, ms ...interface{}) Routes {
	endpoint := r.endpoint

	if r.defaultms != nil && len(r.defaultms) > 0 {
		ms = append(r.defaultms, ms...)
	}

	routes := make([]*whttp.Route, 0)
	if flag&GM_HEAD > 0 {
		// HEAD /{endpoint}
		routes = append(routes, wgo.HEAD("/"+endpoint, r.HandlerByMethod("HEAD"), ms...)...)
	}
	if flag&GM_GET > 0 {
		// GET /{endpoint}/{id}
		path := fmt.Sprintf("/%s/:%s", endpoint, RowkeyKey)
		// Debug("[Builtin]GET %s", path)
		routes = append(routes, wgo.GET(path, r.HandlerByMethod("GET"), ms...)...)
	}
	if flag&GM_LIST > 0 {
		// GET /{endpoint}
		path := fmt.Sprintf("/%s", endpoint)
		routes = append(routes, wgo.GET(path, r.RESTSearch(), ms...)...)
	}
	if flag&GM_POST > 0 {
		// POST /{endpoint}
		path := fmt.Sprintf("/%s", endpoint)
		routes = append(routes, wgo.POST(path, r.HandlerByMethod("POST"), ms...).SetOptions(optionKey(DescKey), "Create")...)
	}
	if flag&GM_DELETE > 0 {
		// DELETE /{endpoint}/{id}
		path := fmt.Sprintf("/%s/:%s", endpoint, RowkeyKey)
		routes = append(routes, wgo.DELETE(path, r.HandlerByMethod("DELETE"), ms...).SetOptions(optionKey(DescKey), "Delete")...)
	}
	if flag&GM_PATCH > 0 {
		// PATCH /{endpoint}/{id}
		path := fmt.Sprintf("/%s/:%s", endpoint, RowkeyKey)
		routes = append(routes, wgo.PATCH(path, r.HandlerByMethod("PATCH"), ms...).SetOptions(optionKey(DescKey), "Update")...)
	}
	if flag&GM_PUT > 0 {
		// PUT /{endpoint}/{id}
		path := fmt.Sprintf("/%s/:%s", endpoint, RowkeyKey)
		routes = append(routes, wgo.PUT(path, r.HandlerByMethod("PUT"), ms...).SetOptions(optionKey(DescKey), "Reset")...)
	}

	// reporting
	if flag&GM_RPT > 0 {
		// POST /{endpoint}/{rpt_tag}
		path := fmt.Sprintf("/%s/:%s", endpoint, RptKey)
		// Debug("[rest.Builtin] path: %s", path)
		routes = append(routes, wgo.GET(path, r.RESTSearch(), ms...)...)
	}
	return Routes{routes}
}

// return handler by method
func (r *REST) HandlerByMethod(method string) wgo.HandlerFunc {
	switch strings.ToUpper(method) {
	case "GET":
		return r.RESTGet()
	case "POST":
		return r.RESTPost()
	case "DELETE":
		return r.RESTDelete()
	case "PATCH":
		return r.RESTPatch()
	case "PUT":
		return r.RESTPut()
	case "HEAD":
		return r.RESTHead()
	default:
		return RESTDeny
	}
}

// Func
func (r *REST) RESTGet() wgo.HandlerFunc {
	model := r.New()
	return func(c *wgo.Context) error {
		rest := GetREST(c)
		m := rest.NewModel(model)
		action := m.(Action)
		defer action.Defer(m)

		if _, err := action.PreGet(m); err != nil {
			c.Warn("PreGet error: %s", err)
			return rest.BadRequest(err)
		} else if _, err := action.WillGet(m); err != nil {
			c.Warn("WillGet error: %s", err)
			return rest.BadRequest(err)
		}

		if ret, err := action.OnGet(m); err != nil {
			c.Warn("OnGet error: %s", err)
			if err == ErrNoRecord {
				return rest.NotFound(err)
			} else {
				return rest.InternalError(err)
			}
		} else if ret0, err := action.DidGet(ret); err != nil {
			c.Warn("DidGet error: %s", err)
			return rest.NotOK(err)
		} else if ret1, err := action.PostGet(ret0); err != nil {
			c.Warn("PostGet error: %s", err)
			return rest.NotOK(err)
		} else {
			return rest.OK(ret1)
		}

	}
}

func (_ *REST) RESTSearch() wgo.HandlerFunc {
	// model := r.New()
	return func(c *wgo.Context) error {
		rest := GetREST(c)
		m := rest.Model()
		action := m.(Action)
		defer action.Defer(m)

		if _, err := action.PreSearch(m); err != nil { // presearch准备条件等
			c.Warn("PreSearch error: %s", err)
			return rest.BadRequest(err)
		} else if _, err := action.WillSearch(m); err != nil {
			c.Warn("WillSearch error: %s", err)
			return rest.BadRequest(err)
		}

		if l, err := action.OnSearch(m); err != nil {
			if err == ErrNoRecord {
				return rest.NotFound(err)
			} else {
				return rest.InternalError(err)
			}
		} else if l0, err := action.DidSearch(l); err != nil {
			c.Warn("DidSearch error: %s", err)
			return rest.NotOK(err)
		} else if rl, err := action.PostSearch(l0); err != nil {
			c.Warn("PostSearch error: %s", err)
			return rest.NotOK(err)
		} else {
			return rest.OK(rl)
		}

	}
}

func (r *REST) RESTPost() wgo.HandlerFunc {
	model := r.New()
	return func(c *wgo.Context) error {
		rest := GetREST(c)
		m := rest.NewModel(model)
		action := m.(Action)
		defer action.Defer(m)

		if _, err := action.PreCreate(m); err != nil { // prepare
			c.Error("PreCreate error: %s", err)
			return rest.BadRequest(err)
		} else if _, err := action.WillCreate(m); err != nil {
			c.Error("WillCreate error: %s", err)
			return rest.BadRequest(err)
		} else if r, err := action.OnCreate(m); err != nil {
			c.Error("OnCreate error: %s", err)
			return rest.NotOK(err)
		} else if r, err := action.DidCreate(r); err != nil {
			c.Error("DidCreate error: %s", err)
			return rest.NotOK(err)
		} else { // all done
			// c.Debug("set rest new: %+v", m)
			if r, err = action.Trigger(r.(Model)); err != nil {
				c.Warn("Trigger error: %s", err)
			}
			if r, err = action.PostCreate(r); err != nil {
				// create ok, return
				c.Warn("PostCreate error: %s", err)
			}
			return rest.OK(r)
		}
	}

}
func (r *REST) RESTPatch() wgo.HandlerFunc {
	model := r.New()
	return func(c *wgo.Context) error { //修改
		rest := GetREST(c)
		m := rest.NewModel(model)
		action := m.(Action)
		defer action.Defer(m)

		if _, err := action.PreUpdate(m); err != nil {
			c.Warn("[RESTPatch]PreUpdate error: %s", err)
			return rest.BadRequest(err)
		} else if _, err := action.WillUpdate(m); err != nil {
			c.Error("[RESTPatch]WillUpdate error: %s", err)
			return rest.BadRequest(err)
		} else if r, err := action.OnUpdate(m); err != nil {
			c.Warn("[RESTPatch]OnUpdate error: %s", err)
			return rest.NotOK(err)
		} else if r, err := action.DidUpdate(r); err != nil {
			c.Error("[RESTPatch]DidUpdate error: %s", err)
			return rest.NotOK(err)
		} else {
			// 触发器
			_, err = action.Trigger(m)
			if err != nil {
				c.Warn("Trigger error: %s", err)
			}

			// update ok
			if r, err = action.PostUpdate(m); err != nil {
				c.Warn("postCreate error: %s", err)
			}

			return rest.OK(r)
		}
	}
}
func (r *REST) RESTPut() wgo.HandlerFunc {
	model := r.New()
	return func(c *wgo.Context) error { //修改
		rest := GetREST(c)
		m := rest.NewModel(model)
		action := m.(Action)
		defer action.Defer(m)

		if _, err := action.PreUpdate(m); err != nil {
			c.Warn("PreUpdate error: %s", err)
			return rest.BadRequest(err)
		} else if r, err := action.OnUpdate(m); err != nil {
			c.Warn("[RESTPut]OnUpdate error: %s", err)
			return rest.NotOK(err)
		} else {
			// 触发器
			_, err = action.Trigger(m)
			if err != nil {
				c.Warn("Trigger error: %s", err)
			}

			// update ok
			if r, err = action.PostUpdate(m); err != nil {
				c.Warn("postCreate error: %s", err)
			}

			return rest.OK(r)
		}
	}
}
func (r *REST) RESTDelete() wgo.HandlerFunc {
	model := r.New()
	return func(c *wgo.Context) error {
		rest := GetREST(c)
		m := rest.NewModel(model)
		action := m.(Action)
		defer action.Defer(m)

		if _, err := action.PreDelete(m); err != nil { // presearch准备条件等
			c.Warn("[RESTDelete]PreDelete error: %s", err)
			return rest.BadRequest(err)
		} else if r, err := action.OnDelete(m); err != nil {
			c.Warn("[RESTDelete]OnDelete error: %s", err)
			return rest.NotOK(err)
		} else {
			r, err = action.PostDelete(m)
			if err != nil {
				c.Warn("postCreate error: %s", err)
			}
			// 触发器
			_, err = action.Trigger(m)
			if err != nil {
				c.Warn("Trigger error: %s", err)
			}
			return rest.OK(r)
		}

	}
}
func (r *REST) RESTHead() wgo.HandlerFunc {
	model := r.New()
	return func(c *wgo.Context) error { //检查字段
		rest := GetREST(c)
		m := rest.NewModel(model)
		action := m.(Action)
		defer action.Defer(m)

		if _, err := action.PreCheck(m); err != nil {
			c.Warn("PreCheck error: %s", err)
			return rest.BadRequest(err)
		}

		if cnt, err := action.OnCheck(m); err != nil {
			c.Warn("OnCheck error: %s", err)
			if err == ErrNoRecord {
				return rest.NotFound(err)
			} else {
				return rest.InternalError(err)
			}
		} else {
			if cnt, _ := action.PostCheck(cnt); cnt.(int64) > 0 {
				return rest.NotOK(nil)
			} else {
				return rest.OK(nil)
			}
		}

	}
}

// 其他路由
// func (rest *REST) Add(method, path string, h wgo.HandlerFunc, ms ...interface{}) Routes {
func (rest *REST) Add(method, path string, opts ...interface{}) Routes {
	h := rest.HandlerByMethod(method)
	ms := rest.defaultms
	if len(opts) > 0 {
		if tmp, ok := opts[0].((func(*wgo.Context) error)); ok {
			h = wgo.HandlerFunc(tmp)
		} else if tmp, ok := opts[0].(wgo.HandlerFunc); ok {
			h = tmp
		}
	}
	if rest.endpoint != "" {
		path = fmt.Sprint("/", rest.endpoint, path)
	}
	if rest.defaultms != nil && len(rest.defaultms) > 0 {
		ms = append(rest.defaultms, ms...)
	}
	//Debug("method: %s, path: %s, model: %v", method, path, rest.Model())
	switch strings.ToUpper(method) {
	case "GET":
		return Routes{wgo.GET(path, h, ms...)}.SetOptions(BaseModelKey, rest.Model()).SetOptions(EndpointKey, rest.endpoint)
	case "POST":
		return Routes{wgo.POST(path, h, ms...)}.SetOptions(BaseModelKey, rest.Model()).SetOptions(EndpointKey, rest.endpoint)
	case "DELETE":
		return Routes{wgo.DELETE(path, h, ms...)}.SetOptions(BaseModelKey, rest.Model()).SetOptions(EndpointKey, rest.endpoint)
	case "PATCH":
		return Routes{wgo.PATCH(path, h, ms...)}.SetOptions(BaseModelKey, rest.Model()).SetOptions(EndpointKey, rest.endpoint)
	case "PUT":
		return Routes{wgo.PUT(path, h, ms...)}.SetOptions(BaseModelKey, rest.Model()).SetOptions(EndpointKey, rest.endpoint)
	case "HEAD":
		return Routes{wgo.HEAD(path, h, ms...)}.SetOptions(BaseModelKey, rest.Model()).SetOptions(EndpointKey, rest.endpoint)
	default:
		return Routes{wgo.GET(path, h, ms...)}.SetOptions(BaseModelKey, rest.Model()).SetOptions(EndpointKey, rest.endpoint)
	}
}

func optionKey(key string) string {
	return fmt.Sprintf("%s:%s", RESTKey, key)
}

// options
func (rs Routes) SetOptions(k string, v interface{}) Routes {
	rs.Routes.SetOptions(optionKey(k), v)
	return rs
}

// skip auth
func (rs Routes) Free() Routes {
	return rs.SetOptions(SKIPAUTH_KEY, true)
}

// inner
func (rs Routes) Inner() Routes {
	return rs.SetOptions(INNERAUTH_KEY, true)
}

// 限制记录access, 毕竟总不能把密码明文记下来吧
func (rs Routes) LimitAccess() Routes {
	return rs.SetOptions(LimitAccessKey, true)
}

// 自定义act
func (rs Routes) CustomAction(act string) Routes {
	return rs.SetOptions(CustomActionKey, act)
}

// desc string
func (rs Routes) Desc(desc string) Routes {
	return rs.SetOptions(DescKey, desc)
}

func (rest *REST) Options(k string) interface{} {
	if c := rest.Context(); c != nil {
		if opt := c.Options(optionKey(k)); opt != nil {
			return opt
		}
	}
	return nil
}

func (rest *REST) IsInner() bool {
	if inner := rest.Options(INNERAUTH_KEY); inner != nil && inner.(bool) == true {
		return true
	}
	return false
}

func (rest *REST) IsFree() bool {
	if skip := rest.Options(SKIPAUTH_KEY); skip != nil && skip.(bool) == true {
		return true
	}
	return false
}
