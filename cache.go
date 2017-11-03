package wgo

import (
	"bytes"
	"fmt"

	"wgo/wcache"
	"wgo/whttp"
)

// wgo online cache
func Cache() MiddlewareFunc {
	cache := wcache.NewCache()
	wcache.SetLogger(wgo)
	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) (err error) {
			switch c.ServerMode() {
			case "rpc", "wrpc", "grpc":
				return next(c) // rpc暂时不需要缓存
			default:
				if opts := c.Options("cache"); opts == nil { // 只有配置了路由的访问会通过
					return next(c)
				} else {
					cacheOpts := opts.(whttp.Options)
					ttl := cacheOpts["ttl"].(int)
					paramString := ""
					if params, ok := cacheOpts["params"].([]string); ok && len(params) > 0 { // 需要缓存的query参数
						ps := bytes.Buffer{}
						for _, n := range params {
							ps.WriteString(c.QueryParam(n) + ",")
						}
						paramString = ps.String()
					}
					headerString := ""
					if headers, ok := cacheOpts["headers"].([]string); ok && len(headers) > 0 { // 需要缓存的header参数
						hs := bytes.Buffer{}
						for _, n := range headers {
							hs.WriteString(c.Request().(whttp.Request).Header().Get(n))
						}
						headerString = hs.String()
					}
					key := fmt.Sprintf("%s:%s:%s:%s:%s:%s", c.Request().(whttp.Request).Method(), c.Request().(whttp.Request).URL().Path(), c.Mux().Engine().Name(), c.Encoding(), paramString, headerString) // 缓存决定因素为method,path,engine,encoding,params,headers
					if res, err := cache.Get([]byte(key)); err == nil {
						c.Info("got key: %s", key)
						res.(whttp.Response).CopyTo(c.response)
						return nil
					} else {
						c.Error("get key(%s) failed: %s", key, err.Error())
					}

					err = next(c)
					if err == nil && !c.NoCache() {
						// success, cache response
						nr := c.Mux().(*whttp.Mux).NewResponse()
						c.Response().(whttp.Response).CopyTo(nr)
						cache.Set([]byte(key), nr, ttl)
						c.Warn("response(%s) cached", key)
					}
					return err
				}
			}
		}
	}
}
