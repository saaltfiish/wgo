package wgo

import (
	// self import
	"wgo/server"
	"wgo/utils"
	"wgo/whttp"
)

type Servers []*server.Server

func (w *WGO) push(s *server.Server) *WGO {
	if w.servers == nil {
		w.servers = make(Servers, 0)
	}
	w.servers = append(w.servers, s)
	return w
}

// get all http/https servers
func AllServers(labels ...string) (ss Servers) { return wgo.AllServers(labels...) }
func (w *WGO) AllServers(labels ...string) (ss Servers) {
	return w.servers
}

// get all http/https servers
func HTTPServers(labels ...string) (ss Servers) { return wgo.HTTPServers(labels...) }
func (w *WGO) HTTPServers(labels ...string) (ss Servers) {
	return wgo.GetServersByMode("http", labels...)
}

// get all rpc servers
func RPCServers(labels ...string) (ss Servers) { return wgo.RPCServers(labels...) }
func (w *WGO) RPCServers(labels ...string) (ss Servers) {
	return wgo.GetServersByMode("rpc", labels...)
}

// get servers by mode
func (w *WGO) GetServersByMode(mode string, labels ...string) (ss Servers) {
	var ms []string
	switch mode {
	case "http", "https", "whttp":
		ms = []string{"http", "https", "whttp"}
	case "rpc", "grpc", "wrpc":
		ms = []string{"rpc", "grpc", "wrpc"}
	}
	for _, s := range w.servers {
		if utils.InSliceIgnorecase(s.Mode(), ms) { // 只要http的
			if len(labels) > 0 {
				if utils.InSliceIgnorecase(s.Name(), labels) {
					ss = append(ss, s)
				}
			} else {
				ss = append(ss, s)
			}
		}
	}
	return
}

// servers
/* {{{ func Use(m ...interface{}) Servers
 * 默认all
 */
func Use(ms ...interface{}) (ss Servers) {
	if ss = wgo.AllServers(); len(ss) > 0 {
		ss.Use(ms...)
	}
	return
}
func (ss Servers) Use(ms ...interface{}) Servers {
	for _, s := range ss {
		if s.Mux() != nil {
			s.Mux().Use(ms...)
		}
	}
	return ss
}

/* }}} */

/* {{{ func NotFound(m ...interface{}) Servers
 * 默认all
 */
func NotFound(h HandlerFunc) (ss Servers) {
	if ss = wgo.AllServers(); len(ss) > 0 {
		ss.NotFound(h)
	}
	return
}
func (ss Servers) NotFound(h HandlerFunc) Servers {
	for _, s := range ss {
		if s.Mux() != nil {
			switch s.Mode() {
			case "http", "https", "whttp":
				s.Mux().(*whttp.Mux).NotFound(handlerFuncToWhttpHandlerFunc(h))
			case "rpc", "wrpc", "grpc":
				Info("wrpc not found")
			default: // do nothing
			}
		}
	}
	return ss
}

/* }}} */
