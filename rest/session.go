// Package rest provides ...
package rest

import (
	"fmt"
	"os"
	"time"

	"wgo"
)

// session

type SessionConfig struct {
	Prefix   string              `json:"prefix"` // 在缓存里的前缀
	Key      string              `json:"key"`    // session key, cookie key
	Life     int                 `json:"life"`   // session life , cookie 过期时间
	Path     string              `json:"path"`
	Domain   string              `json:"domain"`
	Security bool                `json:"security"`
	HTTPOnly bool                `json:"http_only"`
	Redis    []map[string]string `json:"redis"`
}

var scfg *SessionConfig = new(SessionConfig)

// cache key
func cacheKey(key string) (ck string) {
	return fmt.Sprintf("%s:%s", scfg.Prefix, key)
}

// get session
func (rest *REST) Session(opts ...string) (key string, value interface{}) {
	c := rest.Context()
	if s, err := c.Cookie(scfg.Key); err == nil { // 优先从cookie中获取sessionid, 防止客户端的攻击
		key = s.Value()
	} else if len(opts) > 0 && opts[0] != "" { // 传入session key, 主动获取
		key = opts[0]
	} else {
		// 没有获取到key, return
		c.Info("can't get any key about cookie")
		return
	}

	var err error
	if value = rest.GetEnv(SESSION_KEY); value != nil {
		// 内存里找到, 返回
		return
	} else if value, err = RedisGet(cacheKey(key)); value != nil {
		//c.Info("find auth from cookie(%s): %s", scfg.Key, key)
		rest.SaveSession(value)
	} else if acAddr := GetService("ac"); acAddr != "" { // TODO, `ac` should not hardcore
		// check ac addr from env
		if eac := os.Getenv(AECK_AC_ADDR); eac != "" {
			// env overwrite config
			acAddr = eac
		}
		path := "/auth/" + key
		rest.Debug("[Session]ac: %s, query path: %s", acAddr, path)
		if resp, re := NewInnerClient(acAddr).Get(path); re == nil && resp.Code() == 0 && len(resp.Body()) > 0 {
			rest.Debug("[Session]ac response: %d", resp.StatusCode())
			value = resp.Body()
			err = re
			RedisSet(cacheKey(key), value, 86400)
			// rest.SaveSession(value)
		} else {
			rest.Debug("[Session]ac response code: %d", resp.StatusCode())
		}
	} else {
		c.Warn("not found auth by cookie(%s): %s", scfg.Key, err.Error())
	}
	return
}

// save session
func (rest *REST) SaveSession(session interface{}) {
	rest.SetEnv(SESSION_KEY, session)
}

// set session
func (rest *REST) SetSession(key string, opts ...interface{}) {
	// save to cache
	rest.Debug("set session, key: %s, opts: %+v", key, opts)
	RedisSet(cacheKey(key), opts[0], scfg.Life)
	// expire
	expire := time.Time{}
	if len(opts) > 1 {
		if remember, ok := opts[1].(bool); ok && remember {
			expire = time.Now().Add(time.Duration(scfg.Life) * time.Second)
		}
	}
	// set cookie
	rest.Debug("set session, key: %s, path: %s, domain: %s, life: %d", key, scfg.Path, scfg.Domain, scfg.Life)
	rest.Context().SetCookie(wgo.NewCookie(scfg.Key, key, scfg.Path, scfg.Domain, expire, scfg.Security, scfg.HTTPOnly))
}

// del session
func (rest *REST) DelSession(opts ...string) (key string) {
	c := rest.Context()
	if s, err := c.Cookie(scfg.Key); err == nil { // 优先从cookie中获取sessionid
		key = s.Value()
	} else if len(opts) > 0 && opts[0] != "" { // 传入session key, 主动获取
		key = opts[0]
	} else {
		// 没有获取到key, return
		return
	}

	RedisDel(cacheKey(key))
	c.SetCookie(wgo.NewCookie(scfg.Key, "", scfg.Path, scfg.Domain, time.Now().Add(time.Duration(scfg.Life)*time.Second), scfg.Security, scfg.HTTPOnly))

	return
}

// 鉴权+session, 包括cs,ss
func Auth() wgo.MiddlewareFunc {
	if err := wgo.Cfg().AppConfig(scfg, "session"); err != nil {
		Info("not found session scfg")
	}
	if scfg.Prefix == "" {
		scfg.Prefix = "auth"
	}
	if sk := os.Getenv(AECK_SESSION_KEY); sk != "" {
		scfg.Key = sk
	} else if scfg.Key == "" {
		scfg.Key = "asgard"
	}
	if scfg.Life == 0 {
		scfg.Life = 1800
	}
	if scfg.Path == "" {
		scfg.Path = "/"
	}
	if sd := os.Getenv(AECK_SESSION_DOMAIN); sd != "" {
		// 可通过环境参数传入, for local/develop env
		scfg.Domain = sd
	} else if scfg.Domain == "" {
		scfg.Domain = ".gladsheim.cn"
	}
	// get storage
	if tc := os.Getenv(AECK_REDIS_ADDR); tc != "" {
		// 如果环境变量传入, 优先
		scfg.Redis = []map[string]string{map[string]string{"conn": tc, "db": "0"}}
	}
	if len(scfg.Redis) > 0 {
		// 如果设置了redis信息, 则使用(session使用的storage可与底层wgo不同)
		OpenRedis(scfg)
	}
	return func(next wgo.HandlerFunc) wgo.HandlerFunc {
		return func(c *wgo.Context) (err error) {

			// cs用户端访问鉴权
			if k, v := GetREST(c).Session(); k != "" && v != nil {
				c.Authorize() // 授权
			} else {
				// TODO, server端访问鉴权
			}

			return next(c)
		}
	}
}
