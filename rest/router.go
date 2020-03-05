// Package rest provides ...
package rest

import (
	"fmt"
	"net/http"
	"strings"
	"sync"

	"wgo"
	"wgo/server"
	"wgo/whttp"
)

// type Router interface {
// 	GET(*wgo.Context) error
// 	LIST(*wgo.Context) error
// 	POST(*wgo.Context) error
// 	PUT(*wgo.Context) error
// 	DELETE(*wgo.Context) error
// 	PATCH(*wgo.Context) error
// 	HEAD(*wgo.Context) error
// 	OPTIONS(*wgo.Context) error
// 	TRACE(*wgo.Context) error
// }

type Options map[string]interface{}

type Routes struct {
	whttp.Routes
}

// 默认的middleware, 所有rest路由都需要有这两个
var defaultMiddlewares []interface{}

func init() {
	defaultMiddlewares = []interface{}{
		Init(),
		Auth(),
	}
}

func AddMiddleware(ms ...interface{}) {
	defaultMiddlewares = append(defaultMiddlewares, ms...)
	// 检查是否已经有注册rest
	if rp := restPool.Items(); len(rp) > 0 {
		for n, pi := range rp {
			Debug("[AddMiddleware]%s already registered", n)
			r := pi.(*sync.Pool).Get().(*REST)
			// 生成新的pool覆盖
			rest := addREST(r.mg(), r.endpoint, nil, r.flag)
			rest.Builtin(r.flag).SetOptions(ModelPoolKey, rest.Pool())
		}
	}
}

// get router
// 获取默认路由(model名的复数形式)用这个方法, 否则用Register
func GetRouter(i interface{}, opts ...interface{}) *REST {
	r := getREST(i)
	if r == nil {
		// 说明models没有AddModel(i), 这里自动加上
		if m := AddModel(i); m != nil {
			return m.GetREST()
		}
	}
	return r
}

// deny
func RESTDeny(c *wgo.Context) error {
	// rk := c.Param(RowkeyKey)
	// c.Info("[RESTDeny]rowkey: %s", rk)
	return server.NewError(http.StatusMethodNotAllowed, http.StatusText(http.StatusMethodNotAllowed))
}

// 注册路由, 注意AddModel会默认注册了路由, 如果需要改变endpoint, 需要使用Register
func Register(endpoint string, i interface{}, flag int, ms ...interface{}) *REST {
	m, ok := i.(Model)
	if !ok {
		panic("[Register]input not Model")
	}

	if or := getREST(i); or != nil {
		// Info("[Register]rest exists!!!%+v", i)
		if or.Endpoint() != endpoint {
			// endpoint不一致，把旧的builtin路由禁用
			Debug("[Register]new endpoint: %s", endpoint)
			or.Builtin(GM_NONE).SetOptions(ModelPoolKey, or.Pool())
		}
	}

	// 生成新的pool覆盖
	rest := addREST(m, endpoint, ms, flag)
	rest.Builtin(flag).SetOptions(ModelPoolKey, rest.Pool())

	return rest
}

// 内置方法
func (r *REST) Builtin(flag int, ms ...interface{}) Routes {
	endpoint := r.Endpoint()

	if r.defaultms != nil && len(r.defaultms) > 0 {
		ms = append(r.defaultms, ms...)
	}

	routes := make([]*whttp.Route, 0)

	routes = append(routes, wgo.HEAD("/"+endpoint, r.handlerByMethod(GM_HEAD, flag), ms...)...)

	// GET /{endpoint}/{id}
	path := fmt.Sprintf("/%s/:%s", endpoint, RowkeyKey)
	routes = append(routes, wgo.GET(path, r.handlerByMethod(GM_GET, flag), ms...)...)

	// GET /{endpoint}
	path = fmt.Sprintf("/%s", endpoint)
	routes = append(routes, wgo.GET(path, r.handlerByMethod(GM_LIST, flag), ms...)...)

	// POST /{endpoint}
	path = fmt.Sprintf("/%s", endpoint)
	routes = append(routes, wgo.POST(path, r.handlerByMethod(GM_POST, flag), ms...).SetOptions(optionKey(DescKey), "Create")...)

	// DELETE /{endpoint}/{id}
	path = fmt.Sprintf("/%s/:%s", endpoint, RowkeyKey)
	routes = append(routes, wgo.DELETE(path, r.handlerByMethod(GM_DELETE, flag), ms...).SetOptions(optionKey(DescKey), "Delete")...)

	// PATCH /{endpoint}/{id}
	path = fmt.Sprintf("/%s/:%s", endpoint, RowkeyKey)
	routes = append(routes, wgo.PATCH(path, r.handlerByMethod(GM_PATCH, flag), ms...).SetOptions(optionKey(DescKey), "Update")...)

	// PUT /{endpoint}/{id}
	path = fmt.Sprintf("/%s/:%s", endpoint, RowkeyKey)
	routes = append(routes, wgo.PUT(path, r.handlerByMethod(GM_PUT, flag), ms...).SetOptions(optionKey(DescKey), "Reset")...)

	return Routes{routes}
}

// return handler by method
func (r *REST) handlerByMethod(method, flag int) wgo.HandlerFunc {
	if flag&method > 0 {
		switch method {
		case GM_GET:
			return r.RESTGet()
		case GM_POST:
			return r.RESTPost()
		case GM_DELETE:
			return r.RESTDelete()
		case GM_PATCH:
			return r.RESTPatch()
		case GM_PUT:
			return r.RESTPut()
		case GM_HEAD:
			return r.RESTHead()
		case GM_LIST, GM_RPT:
			return r.RESTSearch()
		default:
			return RESTDeny
		}
	}
	return RESTDeny
}

// Func
func (r *REST) RESTGet() wgo.HandlerFunc {
	return func(c *wgo.Context) error {
		rest := GetREST(c)
		m := rest.Model()
		action := m.(Action)
		defer action.Defer()

		if _, err := action.PreGet(); err != nil {
			c.Warn("PreGet error: %s", err)
			return rest.BadRequest(err)
		} else if _, err := action.WillGet(); err != nil {
			c.Warn("WillGet error: %s", err)
			return rest.BadRequest(err)
		}

		if _, err := action.OnGet(); err != nil {
			c.Warn("OnGet error: %s", err)
			if err == ErrNoRecord {
				return rest.NotFound(err)
			} else {
				return rest.InternalError(err)
			}
		} else if _, err := action.DidGet(); err != nil {
			c.Warn("DidGet error: %s", err)
			return rest.NotOK(err)
		} else if ret, err := action.PostGet(); err != nil {
			c.Warn("PostGet error: %s", err)
			return rest.NotOK(err)
		} else {
			return rest.OK(ret)
		}

	}
}

func (_ *REST) RESTSearch() wgo.HandlerFunc {
	return func(c *wgo.Context) error {
		rest := GetREST(c)
		m := rest.Model()
		action := m.(Action)
		defer action.Defer()

		if _, err := action.PreSearch(); err != nil { // presearch准备条件等
			c.Warn("PreSearch error: %s", err)
			return rest.BadRequest(err)
		} else if _, err := action.WillSearch(); err != nil {
			c.Warn("WillSearch error: %s", err)
			return rest.BadRequest(err)
		}

		if _, err := action.OnSearch(); err != nil {
			if err == ErrNoRecord {
				return rest.NotFound(err)
			} else {
				return rest.InternalError(err)
			}
		} else if _, err := action.DidSearch(); err != nil {
			c.Warn("DidSearch error: %s", err)
			return rest.NotOK(err)
		} else if rl, err := action.PostSearch(); err != nil {
			c.Warn("PostSearch error: %s", err)
			return rest.NotOK(err)
		} else {
			return rest.OK(rl)
		}

	}
}

func (r *REST) RESTPost() wgo.HandlerFunc {
	return func(c *wgo.Context) error {
		rest := GetREST(c)
		m := rest.Model()
		action := m.(Action)
		defer action.Defer()

		_, err := action.PreCreate()
		if err != nil { // prepare
			c.Error("PreCreate error: %s", err)
			return rest.BadRequest(err)
		}
		_, err = action.WillCreate()
		if err != nil {
			c.Error("WillCreate error: %s", err)
			return rest.BadRequest(err)
		}
		_, err = action.OnCreate()
		if err != nil {
			c.Error("OnCreate error: %s", err)
			return rest.NotOK(err)
		}
		_, err = action.DidCreate()
		if err != nil {
			c.Error("DidCreate error: %s", err)
			return rest.NotOK(err)
		}
		// all done
		_, err = action.Trigger()
		if err != nil {
			c.Warn("Trigger error: %s", err)
		}
		rt, err := action.PostCreate()
		if err != nil {
			// create ok, return
			c.Warn("PostCreate error: %s", err)
		}
		return rest.OK(rt)
	}

}
func (r *REST) RESTPatch() wgo.HandlerFunc {
	return func(c *wgo.Context) error { //修改
		rest := GetREST(c)
		m := rest.Model()
		action := m.(Action)
		defer action.Defer()

		if _, err := action.PreUpdate(); err != nil {
			c.Warn("[RESTPatch]PreUpdate error: %s", err)
			return rest.BadRequest(err)
		} else if _, err := action.WillUpdate(); err != nil {
			c.Error("[RESTPatch]WillUpdate error: %s", err)
			return rest.BadRequest(err)
		} else if _, err := action.OnUpdate(); err != nil {
			c.Warn("[RESTPatch]OnUpdate error: %s", err)
			return rest.NotOK(err)
		} else if _, err := action.DidUpdate(); err != nil {
			c.Error("[RESTPatch]DidUpdate error: %s", err)
			return rest.NotOK(err)
		} else {
			// 触发器
			_, err = action.Trigger()
			if err != nil {
				c.Warn("Trigger error: %s", err)
			}

			// update ok
			rt, err := action.PostUpdate()
			if err != nil {
				c.Warn("postCreate error: %s", err)
			}

			return rest.OK(rt)
		}
	}
}
func (r *REST) RESTPut() wgo.HandlerFunc {
	return func(c *wgo.Context) error { //修改
		rest := GetREST(c)
		m := rest.Model()
		action := m.(Action)
		defer action.Defer()

		if _, err := action.PreUpdate(); err != nil {
			c.Warn("PreUpdate error: %s", err)
			return rest.BadRequest(err)
		} else if _, err := action.OnUpdate(); err != nil {
			c.Warn("[RESTPut]OnUpdate error: %s", err)
			return rest.NotOK(err)
		} else {
			// 触发器
			_, err = action.Trigger()
			if err != nil {
				c.Warn("Trigger error: %s", err)
			}

			// update ok
			rt, err := action.PostUpdate()
			if err != nil {
				c.Warn("[RESTPut]PostUpdate error: %s", err)
			}

			return rest.OK(rt)
		}
	}
}
func (r *REST) RESTDelete() wgo.HandlerFunc {
	return func(c *wgo.Context) error {
		rest := GetREST(c)
		m := rest.Model()
		action := m.(Action)
		defer action.Defer()

		if _, err := action.PreDelete(); err != nil { // presearch准备条件等
			c.Warn("[RESTDelete]PreDelete error: %s", err)
			return rest.BadRequest(err)
		} else if _, err := action.OnDelete(); err != nil {
			c.Warn("[RESTDelete]OnDelete error: %s", err)
			return rest.NotOK(err)
		} else {
			rt, err := action.PostDelete()
			if err != nil {
				c.Warn("postCreate error: %s", err)
			}
			// 触发器
			_, err = action.Trigger()
			if err != nil {
				c.Warn("Trigger error: %s", err)
			}
			return rest.OK(rt)
		}

	}
}
func (r *REST) RESTHead() wgo.HandlerFunc {
	return func(c *wgo.Context) error { //检查字段
		rest := GetREST(c)
		m := rest.Model()
		action := m.(Action)
		defer action.Defer()

		if _, err := action.PreCheck(); err != nil {
			c.Warn("PreCheck error: %s", err)
			return rest.BadRequest(err)
		}

		if cnt, err := action.OnCheck(); err != nil {
			c.Warn("OnCheck error: %s", err)
			if err == ErrNoRecord {
				return rest.NotFound(err)
			} else {
				return rest.InternalError(err)
			}
		} else if _, err := action.PostCheck(); err != nil {
			return rest.InternalError(err)
		} else {
			if cnt.(int64) > 0 {
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
	var h wgo.HandlerFunc
	ms := rest.defaultms
	if len(opts) > 0 {
		if tmp, ok := opts[0].((func(*wgo.Context) error)); ok {
			h = wgo.HandlerFunc(tmp)
		} else if tmp, ok := opts[0].(wgo.HandlerFunc); ok {
			h = tmp
		}
	}
	if ep := rest.Endpoint(); ep != "" {
		path = fmt.Sprint("/", ep, path)
	}
	if rest.defaultms != nil && len(rest.defaultms) > 0 {
		ms = append(rest.defaultms, ms...)
	}
	//Debug("method: %s, path: %s, model: %v", method, path, rest.Model())
	switch strings.ToUpper(method) {
	case "GET":
		return Routes{wgo.GET(path, h, ms...)}.SetOptions(ModelPoolKey, rest.Pool())
	case "POST":
		return Routes{wgo.POST(path, h, ms...)}.SetOptions(ModelPoolKey, rest.Pool())
	case "DELETE":
		return Routes{wgo.DELETE(path, h, ms...)}.SetOptions(ModelPoolKey, rest.Pool())
	case "PATCH":
		return Routes{wgo.PATCH(path, h, ms...)}.SetOptions(ModelPoolKey, rest.Pool())
	case "PUT":
		return Routes{wgo.PUT(path, h, ms...)}.SetOptions(ModelPoolKey, rest.Pool())
	case "HEAD":
		return Routes{wgo.HEAD(path, h, ms...)}.SetOptions(ModelPoolKey, rest.Pool())
	default:
		return Routes{wgo.GET(path, h, ms...)}.SetOptions(ModelPoolKey, rest.Pool())
	}
}

func (rest *REST) MethodGet(path string, opts ...interface{}) Routes {
	return rest.Add("GET", path, opts...)
}
func (rest *REST) MethodPost(path string, opts ...interface{}) Routes {
	return rest.Add("POST", path, opts...)
}
func (rest *REST) MethodDelete(path string, opts ...interface{}) Routes {
	return rest.Add("DELETE", path, opts...)
}
func (rest *REST) MethodPatch(path string, opts ...interface{}) Routes {
	return rest.Add("PATCH", path, opts...)
}
func (rest *REST) MethodPut(path string, opts ...interface{}) Routes {
	return rest.Add("PUT", path, opts...)
}
func (rest *REST) MethodHead(path string, opts ...interface{}) Routes {
	return rest.Add("HEAD", path, opts...)
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
// 限制routes只能通过内部访问
func (rs Routes) Inner() Routes {
	return rs.SetOptions(INNERAUTH_KEY, true)
}

// 限制记录access, 总不能把密码明文记下来吧...
func (rs Routes) LimitAccess() Routes {
	return rs.SetOptions(LimitAccessKey, true)
}

// 自定义act
// func (rs Routes) CustomAction(act string) Routes {
// 	return rs.SetOptions(CustomActionKey, act)
// }

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
