// Package rest provides ...
package rest

import (
	"os"
	"time"

	"wgo"
	"wgo/environ"
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

// get session
func (rest *REST) Session(opts ...string) (key string, value interface{}) {
	c := rest.Context()
	if s, err := c.Cookie(scfg.Key); err == nil { // 优先从cookie中获取sessionid, 防止客户端的攻击
		key = s.Value()
	} else if len(opts) > 0 && opts[0] != "" { // 传入session key, 主动获取
		key = opts[0]
	} else {
		// 没有获取到key, return
		//c.Info("can't get any key about cookie")
		return
	}

	var err error
	if value, err = RedisGet(key); value != nil {
		//c.Info("find auth from cookie(%s): %s", scfg.Key, key)
		rest.SetEnv(SESSION_KEY, value)
	} else {
		c.Info("not find auth from cookie(%s): %s", scfg.Key, err.Error())
	}
	return
}

// save session
func (rest *REST) SetSession(key string, opts ...interface{}) {
	// save to cache
	rest.Info("set session, key: %s", key)
	RedisSet(key, opts[0], scfg.Life)
	// expire
	expire := time.Time{}
	if len(opts) > 1 {
		if remember, ok := opts[1].(bool); ok && remember {
			expire = time.Now().Add(time.Duration(scfg.Life) * time.Second)
		}
	}
	// set cookie
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

	RedisDel(key)
	c.SetCookie(wgo.NewCookie(scfg.Key, "", scfg.Path, scfg.Domain, time.Now().Add(time.Duration(scfg.Life)*time.Second), scfg.Security, scfg.HTTPOnly))

	return
}

// 鉴权+session, 包括cs,ss
func Auth() wgo.MiddlewareFunc {
	if err := environ.Cfg().AppConfig(scfg, "session"); err != nil {
		Info("not found session scfg")
	}
	if scfg.Prefix == "" {
		scfg.Prefix = "auth"
	}
	if scfg.Key == "" {
		scfg.Key = "asgard"
	}
	if scfg.Life == 0 {
		scfg.Life = 1800
	}
	if scfg.Path == "" {
		scfg.Path = "/"
	}
	if scfg.Domain == "" {
		scfg.Domain = ".gladsheim.cn"
	}
	// env conn string
	if tc := os.Getenv("session.redis.conn"); tc != "" {
		if len(scfg.Redis) > 0 {
			scfg.Redis[0]["conn"] = tc
		} else {
			scfg.Redis = []map[string]string{map[string]string{"conn": tc, "db": "0"}}
		}
	}
	OpenRedis(scfg)
	return func(next wgo.HandlerFunc) wgo.HandlerFunc {
		return func(c *wgo.Context) (err error) {

			// cs用户端访问鉴权
			if k, v := c.Ext().(*REST).Session(); k != "" && v != nil {
				c.Authorize() // 授权
			} else {
				// TODO, server端访问鉴权
			}

			return next(c)
		}
	}
}
