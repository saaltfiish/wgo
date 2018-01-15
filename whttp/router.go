package whttp

import (
	//"reflect"
	"wgo/utils"
	//"wgo/middlewares"
)

type (
	// Router is the registry of all registered routes for an `Server` instance for
	// request matching and URL path parameter parsing.
	Router struct {
		mux    *Mux
		tree   *RouteNode
		routes Routes
		depth  int
	}

	// Route contains a handler and information for matching against requests.
	Route struct {
		Method     string
		Path       string
		Handler    HandlerFunc
		Middleware []*Middleware
		opts       Options

		mux *Mux
	}

	Options map[string]interface{}

	Routes []*Route

	RouteNode struct {
		kind          kind
		label         byte
		prefix        string
		parent        *RouteNode
		children      children
		ppath         string
		pnames        []string
		methodHandler *methodHandler
		methodOptions *methodOptions
		Func          Func
		Opts          Options
	}

	Func interface{}

	kind          uint8
	children      []*RouteNode
	methodHandler struct {
		connect Func
		delete  Func
		get     Func
		head    Func
		options Func
		patch   Func
		post    Func
		put     Func
		trace   Func
	}
	methodOptions struct {
		connect Options
		delete  Options
		get     Options
		head    Options
		options Options
		patch   Options
		post    Options
		put     Options
		trace   Options
	}
)

// Path returns route node path
func (rn *RouteNode) Path() string {
	return rn.ppath
}

func (rn *RouteNode) Names() []string {
	return rn.pnames
}

const (
	skind kind = iota // 静态段
	pkind             // 参数段
	akind             // wildcard段
)

// NewRouter returns a new Router instance.
func NewRouter(m *Mux) *Router {
	return &Router{
		mux: m,
		tree: &RouteNode{
			methodHandler: new(methodHandler),
			methodOptions: new(methodOptions),
		},
		//routes: make(map[string]*Route),
		routes: make([]*Route, 0),
	}
}

func (r *Router) BuildRoutes() {
	// mux notfoudhandler middlewares
	mms := r.mux.middleware
	aumm := make([]string, 0)
	nh := r.mux.notFoundHandler

	// add proxy to notfoundhandler
	proxy := Proxy()
	nh = proxy(nh)

	for i := len(mms) - 1; i >= 0; i-- {
		if !utils.InSliceIgnorecase(mms[i].tag, aumm) { // 每个middleware只生效一次
			nh = mms[i].Func(nh)
			aumm = append(aumm, mms[i].tag)
		}
	}
	r.mux.notFoundHandler = nh

	// routes
	if rs := r.Routes(); len(rs) > 0 {
		for _, rt := range rs {
			//Info("method: %s, path: %s, handler: %s, middlewares: %d", rt.Method, rt.Path, handlerName(rt.Handler), len(rt.Middleware))
			// 不能直接把rt.Handler, rt.Middleware代入下面的func, 由于闭包
			// Chain middleware
			aum := make([]string, 0)
			h := rt.Handler
			ms := rt.Middleware
			for i := len(ms) - 1; i >= 0; i-- {
				//Info("path: %s, tag: %s", rt.Path, ms[i].tag)
				if !utils.InSliceIgnorecase(ms[i].tag, aum) { // 每个middleware只生效一次
					h = ms[i].Func(h)
					aum = append(aum, ms[i].tag)
				}
			}
			r.Add(rt.Method, rt.Path, rt.opts, func(c Context) error {
				return h(c)
			})
		}
	}
}

func (r *Router) Depth() int {
	return r.depth
}

func (r *Router) Routes() Routes {
	return r.routes
}

func (r *Router) AddRoute(route *Route) {
	route.mux = r.mux
	r.routes = append(r.routes, route)
}

// 判断一个路由是否已经存在
func (r *Router) Exists(route *Route) bool {
	if len(r.routes) > 0 {
		mnp := route.Method + route.Path
		for _, rt := range r.routes {
			if rt.Method+rt.Path == mnp {
				return true
			}
		}
	}
	return false
}

func (r *Route) use(ms ...interface{}) {
	r.Middleware = append(r.Middleware, r.mux.Middlewares(ms...)...)
}
func (r *Route) abandon(ms ...interface{}) {
	//Info("[before]route: %s, len: %d", r.Path, len(r.Middleware))
	var mws = make([]*Middleware, 0)
	for _, m1 := range r.Middleware {
		var abandon bool
		for _, m2 := range ms {
			if m1.tag == m2.(*Middleware).tag {
				//Info("abandon route(%s) middleware: %#+v, %#+v", r.Path, m1, m2)
				abandon = true
			}
		}
		if !abandon {
			mws = append(mws, m1)
		}
	}

	r.Middleware = mws
}

// set route options
func (r *Route) setOptions(key string, value interface{}) {
	if r.opts == nil {
		r.opts = make(Options)
	}
	r.opts[key] = value
}

// cache options
// 增加cache配置
// order: ttl, []params, []headers
func (r *Route) cache(opts ...interface{}) {
	ttl := 180 // 默认180秒
	params := []string{}
	headers := []string{}

	ol := len(opts)
	if ol >= 1 {
		if es, ok := opts[0].(int); ok && es > 0 {
			ttl = es
		}
	}
	if ol >= 2 {
		if ps, ok := opts[1].([]string); ok && len(ps) > 0 {
			params = ps
		}
	}
	if ol >= 3 {
		if hs, ok := opts[2].([]string); ok && len(hs) > 0 {
			headers = hs
		}
	}
	cacheOpts := Options{
		"ttl":     ttl,
		"params":  params,
		"headers": headers,
	}
	r.setOptions("cache", cacheOpts)
}

func (rs Routes) Use(ms ...interface{}) Routes {
	for _, r := range rs {
		r.use(ms...)
	}
	return rs
}
func (rs Routes) Abandon(ms ...interface{}) Routes {
	for _, r := range rs {
		r.abandon(ms...)
	}
	return rs
}
func (rs Routes) SetOptions(key string, value interface{}) Routes {
	for _, r := range rs {
		r.setOptions(key, value)
	}
	return rs
}
func (rs Routes) Cache(opts ...interface{}) Routes {
	for _, r := range rs {
		r.cache(opts...)
	}
	return rs
}

// Add registers a new route for method and path with matching handler.
func (r *Router) Add(method, path string, opts Options, h Func) {
	// Validate path
	if path == "" {
		//e.logger.Fatal("path cannot be empty")
		panic("path connot be empty")
	}
	if path[0] != '/' {
		path = "/" + path
	}
	ppath := path        // Pristine path
	pnames := []string{} // Param names

	for i, l := 0, len(path); i < l; i++ {
		if path[i] == ':' {
			j := i + 1

			r.insert(method, path[:i], nil, nil, skind, "", nil)
			for ; i < l && path[i] != '/'; i++ {
			}

			pnames = append(pnames, path[j:i])
			path = path[:j] + path[i:]
			i, l = j, len(path)

			if i == l {
				r.insert(method, path[:i], opts, h, pkind, ppath, pnames)
				return
			}
			r.insert(method, path[:i], nil, nil, pkind, ppath, pnames)
		} else if path[i] == '*' {
			r.insert(method, path[:i], nil, nil, skind, "", nil)
			pnames = append(pnames, "_*")
			r.insert(method, path[:i+1], opts, h, akind, ppath, pnames)
			return
		}
	}

	r.insert(method, path, opts, h, skind, ppath, pnames)
}

func (r *Router) insert(method, path string, opts Options, h Func, t kind, ppath string, pnames []string) {
	l := len(pnames)
	if r.depth < 1 {
		r.depth = l
	}

	cn := r.tree // Current node as root
	if cn == nil {
		panic("server ⇛ invalid method")
	}
	search := path

	for {
		sl := len(search)
		pl := len(cn.prefix)
		l := 0

		// LCP
		max := pl
		if sl < max {
			max = sl
		}
		for ; l < max && search[l] == cn.prefix[l]; l++ {
		}

		if l == 0 {
			// At root node
			cn.label = search[0]
			cn.prefix = search
			if h != nil {
				cn.kind = t
				cn.addHandler(method, h)
				cn.addOptions(method, opts)
				cn.ppath = ppath
				cn.pnames = pnames
			}
		} else if l < pl {
			// Split node
			n := newNode(cn.kind, cn.prefix[l:], cn, cn.children, cn.methodHandler, cn.methodOptions, cn.ppath, cn.pnames)

			// Reset parent node
			cn.kind = skind
			cn.label = cn.prefix[0]
			cn.prefix = cn.prefix[:l]
			cn.children = nil
			cn.methodHandler = new(methodHandler)
			cn.methodOptions = new(methodOptions)
			cn.ppath = ""
			cn.pnames = nil

			cn.addChild(n)

			if l == sl {
				// At parent node
				cn.kind = t
				cn.addHandler(method, h)
				cn.addOptions(method, opts)
				cn.ppath = ppath
				cn.pnames = pnames
			} else {
				// Create child node
				n = newNode(t, search[l:], cn, nil, new(methodHandler), new(methodOptions), ppath, pnames)
				n.addHandler(method, h)
				n.addOptions(method, opts)
				cn.addChild(n)
			}
		} else if l < sl {
			search = search[l:]
			c := cn.findChildWithLabel(search[0])
			if c != nil {
				// Go deeper
				cn = c
				continue
			}
			// Create child node
			n := newNode(t, search, cn, nil, new(methodHandler), new(methodOptions), ppath, pnames)
			n.addHandler(method, h)
			n.addOptions(method, opts)
			cn.addChild(n)
		} else {
			// Node already exists
			if h != nil {
				cn.addHandler(method, h)
				cn.addOptions(method, opts)
				cn.ppath = ppath
				cn.pnames = pnames
			}
		}
		return
	}
}

func newNode(t kind, pre string, p *RouteNode, c children, mh *methodHandler, mopts *methodOptions, ppath string, pnames []string) *RouteNode {
	return &RouteNode{
		kind:          t,
		label:         pre[0],
		prefix:        pre,
		parent:        p,
		children:      c,
		ppath:         ppath,
		pnames:        pnames,
		methodHandler: mh,
		methodOptions: mopts,
	}
}

func (n *RouteNode) addChild(c *RouteNode) {
	n.children = append(n.children, c)
}

func (n *RouteNode) findChild(l byte, t kind) *RouteNode {
	for _, c := range n.children {
		if c.label == l && c.kind == t {
			return c
		}
	}
	return nil
}

func (n *RouteNode) findChildWithLabel(l byte) *RouteNode {
	for _, c := range n.children {
		if c.label == l {
			return c
		}
	}
	return nil
}

func (n *RouteNode) findChildByKind(t kind) *RouteNode {
	for _, c := range n.children {
		if c.kind == t {
			return c
		}
	}
	return nil
}

func (n *RouteNode) addOptions(method string, opts Options) {
	switch method {
	case METHOD_GET:
		n.methodOptions.get = opts
	case METHOD_POST:
		n.methodOptions.post = opts
	case METHOD_PUT:
		n.methodOptions.put = opts
	case METHOD_DELETE:
		n.methodOptions.delete = opts
	case METHOD_PATCH:
		n.methodOptions.patch = opts
	case METHOD_OPTIONS:
		n.methodOptions.options = opts
	case METHOD_HEAD:
		n.methodOptions.head = opts
	case METHOD_CONNECT:
		n.methodOptions.connect = opts
	case METHOD_TRACE:
		n.methodOptions.trace = opts
	}
}

func (n *RouteNode) addHandler(method string, h Func) {
	switch method {
	case METHOD_GET:
		n.methodHandler.get = h
	case METHOD_POST:
		n.methodHandler.post = h
	case METHOD_PUT:
		n.methodHandler.put = h
	case METHOD_DELETE:
		n.methodHandler.delete = h
	case METHOD_PATCH:
		n.methodHandler.patch = h
	case METHOD_OPTIONS:
		n.methodHandler.options = h
	case METHOD_HEAD:
		n.methodHandler.head = h
	case METHOD_CONNECT:
		n.methodHandler.connect = h
	case METHOD_TRACE:
		n.methodHandler.trace = h
	}
}

func (n *RouteNode) findOptions(method string) Options {
	switch method {
	case METHOD_GET:
		return n.methodOptions.get
	case METHOD_POST:
		return n.methodOptions.post
	case METHOD_PUT:
		return n.methodOptions.put
	case METHOD_DELETE:
		return n.methodOptions.delete
	case METHOD_PATCH:
		return n.methodOptions.patch
	case METHOD_OPTIONS:
		return n.methodOptions.options
	case METHOD_HEAD:
		return n.methodOptions.head
	case METHOD_CONNECT:
		return n.methodOptions.connect
	case METHOD_TRACE:
		return n.methodOptions.trace
	default:
		return nil
	}
}

func (n *RouteNode) findHandler(method string) Func {
	switch method {
	case METHOD_GET:
		return n.methodHandler.get
	case METHOD_POST:
		return n.methodHandler.post
	case METHOD_PUT:
		return n.methodHandler.put
	case METHOD_DELETE:
		return n.methodHandler.delete
	case METHOD_PATCH:
		return n.methodHandler.patch
	case METHOD_OPTIONS:
		return n.methodHandler.options
	case METHOD_HEAD:
		return n.methodHandler.head
	case METHOD_CONNECT:
		return n.methodHandler.connect
	case METHOD_TRACE:
		return n.methodHandler.trace
	default:
		return nil
	}
}

//func (n *node) checkMethodNotAllowed() Func {
//	for _, m := range HTTPMethods {
//		if h := n.findHandler(m); h != nil {
//			return MethodNotAllowedHandler
//		}
//	}
//	return NotFoundHandler
//}

// Find lookup a handler registed for method and path. It also parses URL for path
// parameters and load them into context.
func (r *Router) Find(method, path string, pvalues []string) *RouteNode {
	cn := r.tree // Current node as root

	var (
		search = path
		c      *RouteNode // Child node
		n      int        // Param counter
		nk     kind       // Next kind
		nn     *RouteNode // Next node
		ns     string     // Next search
		//pvalues = context.ParamValues()
	)

	// Search order static > param > any
	for {
		if search == "" {
			goto End
		}

		pl := 0 // Prefix length
		l := 0  // LCP length

		if cn.label != ':' {
			sl := len(search)
			pl = len(cn.prefix)

			// LCP
			max := pl
			if sl < max {
				max = sl
			}
			for ; l < max && search[l] == cn.prefix[l]; l++ {
			}
		}

		if l == pl {
			// Continue search
			search = search[l:]
		} else {
			cn = nn
			search = ns
			if nk == pkind {
				goto Param
			} else if nk == akind {
				goto Any
			}
			return nil
		}

		if search == "" {
			goto End
		}

		// Static node
		if c = cn.findChild(search[0], skind); c != nil {
			// Save next
			if cn.label == '/' {
				nk = pkind
				nn = cn
				ns = search
			}
			cn = c
			continue
		}

		// Param node
	Param:
		if c = cn.findChildByKind(pkind); c != nil {
			if len(pvalues) == n {
				continue
			}

			// Save next
			if cn.label == '/' {
				nk = akind
				nn = cn
				ns = search
			}

			cn = c
			i, l := 0, len(search)
			for ; i < l && search[i] != '/'; i++ {
			}
			pvalues[n] = search[:i]
			n++
			search = search[i:]
			continue
		}

		// Any node
	Any:
		if cn = cn.findChildByKind(akind); cn == nil {
			if nn != nil {
				cn = nn
				nn = nil // Next
				search = ns
				if nk == pkind {
					goto Param
				} else if nk == akind {
					goto Any
				}
			}
			// Not found
			return nil
		}
		pvalues[len(cn.pnames)-1] = search
		goto End
	}

End:
	f := cn.findHandler(method)
	o := cn.findOptions(method)

	if f == nil {
		if cn = cn.findChildByKind(akind); cn == nil {
			return cn
		}

		if h := cn.findHandler(method); h != nil {
			cn.Func = h
			cn.Opts = cn.findOptions(method)
		} else {
			cn.Func = nil
			cn.Opts = nil
		}
		pvalues[len(cn.pnames)-1] = ""
	} else {
		cn.Func = f
		cn.Opts = o
	}

	return cn
}
