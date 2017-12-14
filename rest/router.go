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

// 注册路由
// 注册之后可以自动获得rest提供的通用方法,这是rest的核心价值之一
// 同时也可以自己写同名方法覆盖
func Register(endpoint string, i interface{}, flag int, ms ...interface{}) *REST {
	if _, ok := i.(Router); !ok {
		panic("input not Router")
	}
	rt := i.(Router)
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
	wgo.HEAD("/"+endpoint, rt.HEAD)
	wgo.GET("/"+endpoint+"/:"+RowkeyKey, rt.GET)
	wgo.GET("/"+endpoint, rt.LIST)
	wgo.POST("/"+endpoint, rt.POST)
	wgo.DELETE("/"+endpoint+"/:"+RowkeyKey, rt.DELETE)
	wgo.PATCH("/"+endpoint+"/:"+RowkeyKey, rt.PATCH)
	wgo.PUT("/"+endpoint+"/:"+RowkeyKey, rt.PUT)

	rest.Builtin(flag)
	return rest
}

// 内置方法
func (rest *REST) Builtin(flag int, ms ...interface{}) {
	endpoint := rest.endpoint

	if rest.defaultms != nil && len(rest.defaultms) > 0 {
		ms = append(rest.defaultms, ms...)
	}

	if flag&GM_HEAD > 0 {
		// HEAD /{endpoint}
		wgo.HEAD("/"+endpoint, rest.RESTHead(), ms...)
	}
	if flag&GM_GET > 0 {
		// GET /{endpoint}/{id}
		wgo.GET("/"+endpoint+"/:"+RowkeyKey, rest.RESTGet(), ms...)
	}
	if flag&GM_LIST > 0 {
		// GET /{endpoint}
		wgo.GET("/"+endpoint, rest.RESTSearch(), ms...)
	}
	if flag&GM_POST > 0 {
		// POST /{endpoint}
		wgo.POST("/"+endpoint, rest.RESTPost(), ms...)
	}
	if flag&GM_DELETE > 0 {
		// DELETE /{endpoint}/{id}
		wgo.DELETE("/"+endpoint+"/:"+RowkeyKey, rest.RESTDelete(), ms...)
	}
	if flag&GM_PATCH > 0 {
		// PATCH /{endpoint}/{id}
		wgo.PATCH("/"+endpoint+"/:"+RowkeyKey, rest.RESTPatch(), ms...)
	}
	if flag&GM_PUT > 0 {
		// PUT /{endpoint}/{id}
		wgo.PUT("/"+endpoint+"/:"+RowkeyKey, rest.RESTPut(), ms...)
	}
}

// Func
func (r *REST) RESTGet() wgo.HandlerFunc {
	model := r.New()
	return func(c *wgo.Context) error {
		rest := c.Ext().(*REST)
		m := rest.NewModel(model)
		action := m.(Action)
		defer action.Defer(m)

		if _, err := action.PreGet(m); err != nil {
			c.Warn("PreGet error: %s", err)
			return rest.BadRequest(err)
		}

		if ret, err := action.OnGet(m); err != nil {
			c.Warn("OnGet error: %s", err)
			if err == ErrNoRecord {
				return rest.NotFound(err)
			} else {
				return rest.InternalError(err)
			}
		} else if ret1, err := action.PostGet(ret); err != nil {
			c.Warn("PostGet error: %s", err)
			return rest.NotOK(err)
		} else {
			return rest.OK(ret1)
		}

	}
}
func (r *REST) RESTSearch() wgo.HandlerFunc {
	model := r.New()
	return func(c *wgo.Context) error {
		rest := c.Ext().(*REST)
		m := rest.NewModel(model)
		action := m.(Action)
		defer action.Defer(m)

		if _, err := action.PreSearch(m); err != nil { // presearch准备条件等
			c.Warn("PreSearch error: %s", err)
			return rest.BadRequest(err)
		}

		if l, err := action.OnSearch(m); err != nil {
			if err == ErrNoRecord {
				return rest.NotFound(err)
			} else {
				return rest.InternalError(err)
			}
		} else if rl, err := action.PostSearch(l); err != nil {
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
		rest := c.Ext().(*REST)
		m := rest.NewModel(model)
		action := m.(Action)
		defer action.Defer(m)

		if _, err := action.PreCreate(m); err != nil { // presearch准备条件等
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
			return rest.BadRequest(err)
		} else { // 已经创建成功, 返回成功
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
		rest := c.Ext().(*REST)
		m := rest.NewModel(model)
		action := m.(Action)
		defer action.Defer(m)

		if _, err := action.PreUpdate(m); err != nil {
			c.Warn("PreUpdate error: %s", err)
			return rest.BadRequest(err)
		} else if r, err := action.OnUpdate(m); err != nil {
			c.Warn("OnUpdate error: %s", err)
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
		rest := c.Ext().(*REST)
		m := rest.NewModel(model)
		action := m.(Action)
		defer action.Defer(m)

		if _, err := action.PreUpdate(m); err != nil {
			c.Warn("PreUpdate error: %s", err)
			return rest.BadRequest(err)
		} else if r, err := action.OnUpdate(m); err != nil {
			c.Warn("OnUpdate error: %s", err)
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
		rest := c.Ext().(*REST)
		m := rest.NewModel(model)
		action := m.(Action)
		defer action.Defer(m)

		if _, err := action.PreDelete(m); err != nil { // presearch准备条件等
			c.Warn("PreUpdat error: %s", err)
			return rest.BadRequest(err)
		} else if r, err := action.OnDelete(m); err != nil {
			c.Warn("OnUpdate error: %s", err)
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
		rest := c.Ext().(*REST)
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
func (rest *REST) Add(method, path string, h wgo.HandlerFunc, ms ...interface{}) Routes {
	if rest.endpoint != "" {
		path = fmt.Sprint("/", rest.endpoint, path)
	}
	if rest.defaultms != nil && len(rest.defaultms) > 0 {
		ms = append(rest.defaultms, ms...)
	}
	//Debug("method: %s, path: %s, model: %v", method, path, rest.Model())
	switch strings.ToUpper(method) {
	case "GET":
		return Routes{wgo.GET(path, h, ms...)}
	case "POST":
		return Routes{wgo.POST(path, h, ms...)}
	case "DELETE":
		return Routes{wgo.DELETE(path, h, ms...)}
	case "PATCH":
		return Routes{wgo.PATCH(path, h, ms...)}
	case "PUT":
		return Routes{wgo.PUT(path, h, ms...)}
	case "HEAD":
		return Routes{wgo.HEAD(path, h, ms...)}
	default:
		return Routes{wgo.GET(path, h, ms...)}
	}
}

// options
func (rs Routes) SetOptions(k string, v interface{}) {
	rs.Routes.SetOptions("rest", whttp.Options{k: v})
}

func (rest *REST) Options(k string) interface{} {
	if c := rest.Context(); c != nil {
		if opts := c.Options("rest"); opts != nil {
			if opt, ok := opts.(whttp.Options)[k]; ok {
				return opt
			}
		}
	}
	return nil
}
