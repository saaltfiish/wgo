package wgo

import (
	"path"
	"reflect"

	// self import
	"wgo/server"
	"wgo/whttp"
)

type (
	// HTTPGroup 一个前缀下的所有路由
	HTTPGroup struct {
		prefix string
		//middleware  []whttp.MiddlewareFunc
		middleware []*whttp.Middleware
		engine     server.Engine
	}
	// HTTPGroups 多个HTTPGroup
	HTTPGroups []*HTTPGroup
)

// 把MiddlewareFunc or whttp.MiddlewareFunc 转换为 whttp.MiddlewareFunc
func mixWhttpMiddlewares(ms ...interface{}) []*whttp.Middleware {
	// 支持whttp.Middleware, Middleware两种中间件
	wms := []*whttp.Middleware{}
	if len(ms) > 0 {
		for _, m := range ms {
			if _, ok := m.(MiddlewareFunc); ok {
				wms = append(wms, newWhttpMiddleware(m.(MiddlewareFunc)))
			} else if _, ok := m.(whttp.MiddlewareFunc); ok {
				wms = append(wms, whttp.NewMiddleware(m.(whttp.MiddlewareFunc).Name(), m.(whttp.MiddlewareFunc)))
			}
		}
	}
	return wms
}

// HandlerFunc to whttp.HandlerFunc
func handlerFuncToWhttpHandlerFunc(h HandlerFunc) whttp.HandlerFunc {
	return func(c whttp.Context) error {
		return h(c.(*Context))
	}
}

// whttp.HandlerFunc to HandlerFunc
func whttpHandlerFuncToHandlerFunc(h whttp.HandlerFunc) HandlerFunc {
	return func(c *Context) error {
		return h(c)
	}
}

// Middleware to whttp.Middleware
func middlewareToWhttpMiddleware(m MiddlewareFunc) whttp.MiddlewareFunc {
	return func(h whttp.HandlerFunc) whttp.HandlerFunc {
		return handlerFuncToWhttpHandlerFunc(m(whttpHandlerFuncToHandlerFunc(h)))
	}
}

// new whttp middleware
func newWhttpMiddleware(m MiddlewareFunc) *whttp.Middleware {
	return whttp.NewMiddleware(m.Name(), middlewareToWhttpMiddleware(m))
}

// whttp.Middleware to Middleware
func whttpMiddlewareToMiddleware(m whttp.MiddlewareFunc) MiddlewareFunc {
	return func(h HandlerFunc) HandlerFunc {
		return whttpHandlerFuncToHandlerFunc(m(handlerFuncToWhttpHandlerFunc(h)))
	}
}

func NewGroup(eng server.Engine, prefix string, m ...interface{}) (g *HTTPGroup) {
	g = &HTTPGroup{prefix: prefix, engine: eng}
	g.use(m...)
	return
}

func (g *HTTPGroup) use(ms ...interface{}) {
	g.middleware = append(g.middleware, mixWhttpMiddlewares(ms...)...)
}

func (g *HTTPGroup) abandon(ms ...interface{}) {
	wms := []*whttp.Middleware{}
	for _, m1 := range g.middleware {
		var abandon bool
		for _, m2 := range ms {
			if reflect.ValueOf(m1).Pointer() == reflect.ValueOf(m2).Pointer() {
				//Info("abandon group middleware: %q, %q", m1, m2)
				abandon = true
			}
		}
		if !abandon {
			wms = append(wms, m1)
		}
	}

	g.middleware = wms
}

func (g *HTTPGroup) add(method, path string, h HandlerFunc, ms ...interface{}) *whttp.Route {
	return g.engine.Mux().(*whttp.Mux).Add(method, g.prefix+path, handlerFuncToWhttpHandlerFunc(h), ms...)
}

// groups
func (gs HTTPGroups) Use(m ...interface{}) HTTPGroups {
	for _, g := range gs {
		g.use(m...)
	}
	return gs
}

func (gs HTTPGroups) Abandon(m ...interface{}) HTTPGroups {
	for _, g := range gs {
		g.abandon(m...)
	}
	return gs
}

func (gs HTTPGroups) CONNECT(path string, h HandlerFunc, ms ...interface{}) whttp.Routes {
	rs := make([]*whttp.Route, 0)
	for _, g := range gs {
		r := g.add(whttp.METHOD_CONNECT, path, h, ms...)
		rs = append(rs, r)
	}
	return rs
}

func (gs HTTPGroups) DELETE(path string, h HandlerFunc, ms ...interface{}) whttp.Routes {
	rs := make([]*whttp.Route, 0)
	for _, g := range gs {
		r := g.add(whttp.METHOD_DELETE, path, h, ms...)
		rs = append(rs, r)
	}
	return rs
}

func (gs HTTPGroups) GET(path string, h HandlerFunc, ms ...interface{}) whttp.Routes {
	rs := make([]*whttp.Route, 0)
	for _, g := range gs {
		r := g.add(whttp.METHOD_GET, path, h, ms...)
		rs = append(rs, r)
	}
	return rs
}

func (gs HTTPGroups) HEAD(path string, h HandlerFunc, ms ...interface{}) whttp.Routes {
	rs := make([]*whttp.Route, 0)
	for _, g := range gs {
		r := g.add(whttp.METHOD_HEAD, path, h, ms...)
		rs = append(rs, r)
	}
	return rs
}

func (gs HTTPGroups) OPTIONS(path string, h HandlerFunc, ms ...interface{}) whttp.Routes {
	rs := make([]*whttp.Route, 0)
	for _, g := range gs {
		r := g.add(whttp.METHOD_OPTIONS, path, h, ms...)
		rs = append(rs, r)
	}
	return rs
}

func (gs HTTPGroups) PATCH(path string, h HandlerFunc, ms ...interface{}) whttp.Routes {
	rs := make([]*whttp.Route, 0)
	for _, g := range gs {
		r := g.add(whttp.METHOD_PATCH, path, h, ms...)
		rs = append(rs, r)
	}
	return rs
}

func (gs HTTPGroups) POST(path string, h HandlerFunc, ms ...interface{}) whttp.Routes {
	rs := make([]*whttp.Route, 0)
	for _, g := range gs {
		r := g.add(whttp.METHOD_POST, path, h, ms...)
		rs = append(rs, r)
	}
	return rs
}

func (gs HTTPGroups) PUT(path string, h HandlerFunc, ms ...interface{}) whttp.Routes {
	rs := make([]*whttp.Route, 0)
	for _, g := range gs {
		r := g.add(whttp.METHOD_PUT, path, h, ms...)
		rs = append(rs, r)
	}
	return rs
}

func (gs HTTPGroups) TRACE(path string, h HandlerFunc, ms ...interface{}) whttp.Routes {
	rs := make([]*whttp.Route, 0)
	for _, g := range gs {
		r := g.add(whttp.METHOD_TRACE, path, h, ms...)
		rs = append(rs, r)
	}
	return rs
}

func (gs HTTPGroups) Any(path string, h HandlerFunc, ms ...interface{}) whttp.Routes {
	rs := make([]*whttp.Route, 0)
	for _, g := range gs {
		ars := make([]*whttp.Route, 0)
		for _, method := range whttp.Methods {
			r := g.add(method, path, h, ms...)
			ars = append(ars, r)
		}
		ars = append(rs, ars...)
	}
	return rs
}

func (gs HTTPGroups) Match(methods []string, path string, h HandlerFunc, ms ...interface{}) whttp.Routes {
	rs := make([]*whttp.Route, 0)
	for _, g := range gs {
		mrs := make([]*whttp.Route, 0)
		for _, method := range methods {
			r := g.add(method, path, h, ms...)
			mrs = append(mrs, r)
		}
		rs = append(rs, mrs...)
	}
	return rs
}

func (gs HTTPGroups) Static(prefix, root string) HTTPGroups {
	for _, g := range gs {
		g.engine.Mux().(*whttp.Mux).Add(whttp.METHOD_GET, prefix+"*", handlerFuncToWhttpHandlerFunc(func(c *Context) error {
			return c.File(path.Join(root, c.P(0)))
		}))
	}
	return gs
}

func (gs HTTPGroups) File(path, file string) HTTPGroups {
	for _, g := range gs {
		g.engine.Mux().(*whttp.Mux).Add(whttp.METHOD_GET, path, handlerFuncToWhttpHandlerFunc(func(c *Context) error {
			return c.File(file)
		}))
	}
	return gs
}

/* {{{ func Group(prefix string, m ...interface{}) (gs HTTPGroups)
 * 默认all
 */
func Group(prefix string, ms ...interface{}) (gs HTTPGroups) {
	if ss := wgo.HTTPServers(); len(ss) > 0 {
		return ss.Group(prefix, ms...)
	}
	return
}
func (ss Servers) Group(prefix string, ms ...interface{}) (gs HTTPGroups) {
	gs = make(HTTPGroups, 0)
	for _, s := range ss {
		gs = append(gs, NewGroup(s.Engine().(server.Engine), prefix, ms...))
	}
	return gs
}

/* }}} */

/* {{{ func Abandon(m ...interface{}) Servers
 * 默认all
 */
func Abandon(ms ...interface{}) (ss Servers) {
	if ss = wgo.HTTPServers(); len(ss) > 0 {
		ss.Abandon(ms...)
	}
	return
}
func (ss Servers) Abandon(ms ...interface{}) Servers {
	for _, s := range ss {
		if s.Mux() != nil {
			s.Mux().(*whttp.Mux).Abandon(ms...)
		}
	}

	return ss
}

/* }}} */

/* {{{ func CONNECT(path string, h HandlerFunc, ms ...interface{}) whttp.Routes
 * 默认all
 */
func CONNECT(path string, h HandlerFunc, ms ...interface{}) whttp.Routes {
	if ss := wgo.HTTPServers(); len(ss) > 0 {
		return ss.CONNECT(path, h, ms...)
	}
	return nil
}
func (ss Servers) CONNECT(path string, h HandlerFunc, ms ...interface{}) whttp.Routes {
	rs := make([]*whttp.Route, 0)
	for _, s := range ss {
		if s.Mux() != nil {
			r := s.Mux().(*whttp.Mux).Add(whttp.METHOD_CONNECT, path, handlerFuncToWhttpHandlerFunc(h), ms...)
			rs = append(rs, r)
		}
	}
	return rs
}

/* }}} */

/* {{{ func DELETE(path string, h HandlerFunc, ms ...interface{}) whttp.Routes
 * 默认all
 */
func DELETE(path string, h HandlerFunc, ms ...interface{}) whttp.Routes {
	if ss := wgo.HTTPServers(); len(ss) > 0 {
		return ss.DELETE(path, h, ms...)
	}
	return nil
}
func (ss Servers) DELETE(path string, h HandlerFunc, ms ...interface{}) whttp.Routes {
	rs := make([]*whttp.Route, 0)
	for _, s := range ss {
		if s.Mux() != nil {
			r := s.Mux().(*whttp.Mux).Add(whttp.METHOD_DELETE, path, handlerFuncToWhttpHandlerFunc(h), ms...)
			rs = append(rs, r)
		}
	}
	return rs
}

/* }}} */

/* {{{ func GET(path string, h HandlerFunc, ms ...interface{}) whttp.Routes
 * 默认all
 */
func GET(path string, h HandlerFunc, ms ...interface{}) whttp.Routes {
	if ss := wgo.HTTPServers(); len(ss) > 0 {
		return ss.GET(path, h, ms...)
	}
	return nil
}
func (ss Servers) GET(path string, h HandlerFunc, ms ...interface{}) whttp.Routes {
	rs := make([]*whttp.Route, 0)
	for _, s := range ss {
		if s.Mux() != nil {
			r := s.Mux().(*whttp.Mux).Add(whttp.METHOD_GET, path, handlerFuncToWhttpHandlerFunc(h), ms...)
			rs = append(rs, r)
		}
	}
	return rs
}

/* }}} */

/* {{{ func HEAD(path string, h HandlerFunc, ms ...interface{}) whttp.Routes
 *
 */
func HEAD(path string, h HandlerFunc, ms ...interface{}) whttp.Routes {
	if ss := wgo.HTTPServers(); len(ss) > 0 {
		return ss.HEAD(path, h, ms...)
	}
	return nil
}
func (ss Servers) HEAD(path string, h HandlerFunc, ms ...interface{}) whttp.Routes {
	rs := make([]*whttp.Route, 0)
	for _, s := range ss {
		if s.Mux() != nil {
			r := s.Mux().(*whttp.Mux).Add(whttp.METHOD_HEAD, path, handlerFuncToWhttpHandlerFunc(h), ms...)
			rs = append(rs, r)
		}
	}
	return rs
}

/* }}} */

/* {{{ func OPTIONS(path string, h HandlerFunc, ms ...interface{}) whttp.Routes
 *
 */
func OPTIONS(path string, h HandlerFunc, ms ...interface{}) whttp.Routes {
	if ss := wgo.HTTPServers(); len(ss) > 0 {
		return ss.OPTIONS(path, h, ms...)
	}
	return nil
}
func (ss Servers) OPTIONS(path string, h HandlerFunc, ms ...interface{}) whttp.Routes {
	rs := make([]*whttp.Route, 0)
	for _, s := range ss {
		if s.Mux() != nil {
			//r := s.Engine().(server.Engine).Mux().Add(whttp.METHOD_OPTIONS, path, h, m...)
			r := s.Mux().(*whttp.Mux).Add(whttp.METHOD_OPTIONS, path, handlerFuncToWhttpHandlerFunc(h), ms...)
			rs = append(rs, r)
		}
	}
	return rs
}

/* }}} */

/* {{{ func PATCH(path string, h HandlerFunc, ms ...interface{}) whttp.Routes
 *
 */
func PATCH(path string, h HandlerFunc, ms ...interface{}) whttp.Routes {
	if ss := wgo.HTTPServers(); len(ss) > 0 {
		return ss.PATCH(path, h, ms...)
	}
	return nil
}
func (ss Servers) PATCH(path string, h HandlerFunc, ms ...interface{}) whttp.Routes {
	rs := make([]*whttp.Route, 0)
	for _, s := range ss {
		if s.Mux() != nil {
			r := s.Mux().(*whttp.Mux).Add(whttp.METHOD_PATCH, path, handlerFuncToWhttpHandlerFunc(h), ms...)
			rs = append(rs, r)
		}
	}
	return rs
}

/* }}} */

/* {{{ func POST(path string, h HandlerFunc, ms ...interface{}) whttp.Routes
 *
 */
func POST(path string, h HandlerFunc, ms ...interface{}) whttp.Routes {
	if ss := wgo.HTTPServers(); len(ss) > 0 {
		return ss.POST(path, h, ms...)
	}
	return nil
}
func (ss Servers) POST(path string, h HandlerFunc, ms ...interface{}) whttp.Routes {
	rs := make([]*whttp.Route, 0)
	for _, s := range ss {
		if s.Mux() != nil {
			r := s.Mux().(*whttp.Mux).Add(whttp.METHOD_POST, path, handlerFuncToWhttpHandlerFunc(h), ms...)
			rs = append(rs, r)
		}
	}
	return rs
}

/* }}} */

/* {{{ func PUT(path string, h HandlerFunc, ms ...interface{}) whttp.Routes
 *
 */
func PUT(path string, h HandlerFunc, ms ...interface{}) whttp.Routes {
	if ss := wgo.HTTPServers(); len(ss) > 0 {
		return ss.PUT(path, h, ms...)
	}
	return nil
}
func (ss Servers) PUT(path string, h HandlerFunc, ms ...interface{}) whttp.Routes {
	rs := make([]*whttp.Route, 0)
	for _, s := range ss {
		if s.Mux() != nil {
			r := s.Mux().(*whttp.Mux).Add(whttp.METHOD_PUT, path, handlerFuncToWhttpHandlerFunc(h), ms...)
			rs = append(rs, r)
		}
	}
	return rs
}

/* }}} */

/* {{{ func TRACE(path string, h HandlerFunc, ms ...interface{}) whttp.Routes
 *
 */
func TRACE(path string, h HandlerFunc, ms ...interface{}) whttp.Routes {
	if ss := wgo.HTTPServers(); len(ss) > 0 {
		return ss.TRACE(path, h, ms...)
	}
	return nil
}
func (ss Servers) TRACE(path string, h HandlerFunc, ms ...interface{}) whttp.Routes {
	rs := make([]*whttp.Route, 0)
	for _, s := range ss {
		if s.Mux() != nil {
			r := s.Mux().(*whttp.Mux).Add(whttp.METHOD_TRACE, path, handlerFuncToWhttpHandlerFunc(h), ms...)
			rs = append(rs, r)
		}
	}
	return rs
}

/* }}} */

/* {{{ func Any(path string, h HandlerFunc, ms ...interface{}) whttp.Routes
 *
 */
func Any(path string, h HandlerFunc, ms ...interface{}) whttp.Routes {
	if ss := wgo.HTTPServers(); len(ss) > 0 {
		return ss.Any(path, h, ms...)
	}
	return nil
}
func (ss Servers) Any(path string, h HandlerFunc, ms ...interface{}) whttp.Routes {
	rs := make([]*whttp.Route, 0)
	for _, s := range ss {
		if s.Mux() != nil {
			for _, method := range whttp.Methods {
				r := s.Mux().(*whttp.Mux).Add(method, path, handlerFuncToWhttpHandlerFunc(h), ms...)
				rs = append(rs, r)
			}
		}
	}
	return rs
}

/* }}} */

/* {{{ func Match(methods []string, path string, h HandlerFunc, ms ...interface{}) whttp.Routes
 *
 */
func Match(methods []string, path string, h HandlerFunc, ms ...interface{}) whttp.Routes {
	if ss := wgo.HTTPServers(); len(ss) > 0 {
		return ss.Match(methods, path, h, ms...)
	}
	return nil
}
func (ss Servers) Match(methods []string, path string, h HandlerFunc, ms ...interface{}) whttp.Routes {
	rs := make([]*whttp.Route, 0)
	for _, s := range ss {
		if s.Mux() != nil {
			for _, method := range methods {
				r := s.Mux().(*whttp.Mux).Add(method, path, handlerFuncToWhttpHandlerFunc(h), ms...)
				rs = append(rs, r)
			}
		}
	}
	return rs
}

/* }}} */

/* {{{ func File(path, file string) whttp.Routes
 *
 */
func File(path, file string) whttp.Routes {
	if ss := wgo.HTTPServers(); len(ss) > 0 {
		return ss.File(path, file)
	}
	return nil
}
func (ss Servers) File(path, file string) whttp.Routes {
	rs := make([]*whttp.Route, 0)
	for _, s := range ss {
		if s.Mux() != nil {
			r := s.Mux().(*whttp.Mux).Add(whttp.METHOD_GET, path, func(c whttp.Context) error {
				return c.File(file)
			})
			rs = append(rs, r)
		}
	}
	return rs
}

/* }}} */

/* {{{ func Static(prefix, root string) whttp.Routes
 *
 */
func Static(prefix, root string) whttp.Routes {
	if ss := wgo.HTTPServers(); len(ss) > 0 {
		return ss.Static(prefix, root)
	}
	return nil
}
func (ss Servers) Static(prefix, root string) whttp.Routes {
	rs := make([]*whttp.Route, 0)
	for _, s := range ss {
		if s.Mux() != nil {
			r := s.Mux().(*whttp.Mux).Add(whttp.METHOD_GET, prefix+"*", func(c whttp.Context) error {
				return c.(whttp.Context).File(path.Join(root, c.P(0)))
			})
			rs = append(rs, r)
		}
	}
	return rs
}

/* }}} */
