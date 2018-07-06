package redis

import (
	"encoding/json"
	"errors"
	//"fmt"
	"strconv"
	"time"

	"github.com/garyburd/redigo/redis"

	"wgo/storage/core"
)

var (
	// DefaultKey the collection name of redis for cache adapter.
	DefaultKey = "storage_redis"
)

// Cache is Redis cache adapter.
type Cache struct {
	p        *redis.Pool // redis connection pool
	conninfo string
	dbNum    int
	key      string
	password string
	prefix   string
}

// NewRedisCache create new redis cache with default collection name.
func NewRedisCache() core.Cache {
	return &Cache{key: DefaultKey}
}

// actually do the redis cmds
func (rc *Cache) do(commandName string, args ...interface{}) (reply interface{}, err error) {
	c := rc.p.Get()
	defer c.Close()

	return c.Do(commandName, args...)
}

// Get cache from redis.
func (rc *Cache) Get(key string) interface{} {
	realKey := rc.prefix + key
	if v, err := rc.do("GET", realKey); err == nil {
		return v
	}
	return nil
}

// Get and Set
func (rc *Cache) GetSet(key string, value interface{}) (interface{}, error) {
	realKey := rc.prefix + key
	return rc.do("GETSET", realKey, value)
}

// Get cache from redis.
func (rc *Cache) HGet(key string, field string) interface{} {
	realKey := rc.prefix + key
	if field != "" {
		if v, err := rc.do("HGET", realKey, field); err == nil {
			return v
		}
	} else {
		if v, err := rc.do("HGETALL", realKey); err == nil {
			return v
		}
	}
	return nil
}

// GetMulti get cache from redis.
func (rc *Cache) GetMulti(keys []string) []interface{} {
	size := len(keys)
	var rv []interface{}
	c := rc.p.Get()
	defer c.Close()
	var err error
	for _, key := range keys {
		realKey := rc.prefix + key
		err = c.Send("GET", realKey)
		if err != nil {
			goto ERROR
		}
	}
	if err = c.Flush(); err != nil {
		goto ERROR
	}
	for i := 0; i < size; i++ {
		if v, err := c.Receive(); err == nil {
			rv = append(rv, v.([]byte))
		} else {
			rv = append(rv, err)
		}
	}
	return rv
ERROR:
	rv = rv[0:0]
	for i := 0; i < size; i++ {
		rv = append(rv, nil)
	}

	return rv
}

// Put put key/value to redis. nx 代表只有不存在时才set = setnx
func (rc *Cache) Put(key string, val interface{}, timeout time.Duration, opts ...interface{}) error {
	var err error
	realKey := rc.prefix + key
	cmd := "SET"
	args := []interface{}{
		realKey,
		val,
		"NX", int64(timeout / time.Second),
	}
	if len(opts) > 0 {
		if nx, ok := opts[0].(bool); ok && nx == true {
			args = append(args, "NX")
		}
	}
	// if _, err = rc.do("SETEX", realKey, int64(timeout/time.Second), val); err != nil {
	// 	//fmt.Printf("key: %s, realkey: %s, put failed: %s\n", key, realKey, err.Error())
	// 	return err
	// }
	_, err = rc.do(cmd, args...)
	return err
}

// Delete delete cache in redis.
func (rc *Cache) Delete(key string) error {
	var err error
	realKey := rc.prefix + key
	if _, err = rc.do("DEL", realKey); err != nil {
		return err
	}
	//_, err = rc.do("HDEL", rc.key, realKey)
	return err
}

// IsExist check cache's existence in redis.
func (rc *Cache) IsExist(key string) bool {
	realKey := rc.prefix + key
	v, err := redis.Bool(rc.do("EXISTS", realKey))
	if err != nil {
		return false
	}
	//if v == false {
	//	if _, err = rc.do("HDEL", rc.key, realKey); err != nil {
	//		return false
	//	}
	//}
	return v
}

// Incr increase counter in redis.
func (rc *Cache) Incr(key string) (int, error) {
	realKey := rc.prefix + key
	data, err := redis.Int(rc.do("INCRBY", realKey, 1))
	return data, err
}

// Decr decrease counter in redis.
func (rc *Cache) Decr(key string) (int, error) {
	realKey := rc.prefix + key
	data, err := redis.Int(rc.do("DECRBY", realKey, 1))
	return data, err
}

// Incr increase counter in redis.
func (rc *Cache) IncrBy(key string, num int) (int, error) {
	realKey := rc.prefix + key
	data, err := redis.Int(rc.do("INCRBY", realKey, num))
	return data, err
}

// Decr decrease counter in redis.
func (rc *Cache) DecrBy(key string, num int) (int, error) {
	realKey := rc.prefix + key
	data, err := redis.Int(rc.do("DECRBY", realKey, num))
	return data, err
}

func (rc *Cache) Push(key string, val interface{}) error {
	realKey := rc.prefix + key
	_, err := rc.do("RPUSH", realKey, val)
	return err
}

func (rc *Cache) Pop(key string) (interface{}, error) {
	realKey := rc.prefix + key
	if v, err := rc.do("LPOP", realKey); err == nil {
		return v, nil
	}
	return nil, errors.New("pop error")
}

// ClearAll clean all cache in redis. delete this redis collection.
func (rc *Cache) ClearAll() error {
	cachedKeys, err := redis.Strings(rc.do("HKEYS", rc.key))
	if err != nil {
		return err
	}
	for _, str := range cachedKeys {
		if _, err = rc.do("DEL", str); err != nil {
			return err
		}
	}
	_, err = rc.do("DEL", rc.key)
	return err
}

func (rc *Cache) GetPrefix() string {
	return rc.prefix
}

// Start start redis cache adapter.
// config is like {"key":"collection key","conn":"connection info","dbNum":"0"}
// the cache item in redis are stored forever,
// so no gc operation.
func (rc *Cache) Start(config string) error {
	var cf map[string]string
	json.Unmarshal([]byte(config), &cf)

	if _, ok := cf["key"]; !ok {
		cf["key"] = DefaultKey
	}
	if _, ok := cf["conn"]; !ok {
		return errors.New("config has no conn key")
	}
	if _, ok := cf["dbNum"]; !ok {
		cf["dbNum"] = "0"
	}
	if _, ok := cf["password"]; !ok {
		cf["password"] = ""
	}
	if _, ok := cf["prefix"]; !ok {
		cf["prefix"] = ""
	}

	rc.key = cf["key"]
	rc.conninfo = cf["conn"]
	rc.dbNum, _ = strconv.Atoi(cf["dbNum"])
	rc.password = cf["password"]
	rc.prefix = cf["prefix"]

	rc.connectInit()

	c := rc.p.Get()
	defer c.Close()

	return c.Err()
}

// connect to redis.
func (rc *Cache) connectInit() {
	dialFunc := func() (c redis.Conn, err error) {
		c, err = redis.Dial("tcp", rc.conninfo)
		if err != nil {
			return nil, err
		}

		if rc.password != "" {
			if _, err := c.Do("AUTH", rc.password); err != nil {
				c.Close()
				return nil, err
			}
		}

		_, selecterr := c.Do("SELECT", rc.dbNum)
		if selecterr != nil {
			c.Close()
			return nil, selecterr
		}
		return
	}
	// initialize a new pool
	rc.p = &redis.Pool{
		MaxIdle:     80,
		MaxActive:   100,
		IdleTimeout: 180 * time.Second,
		Dial:        dialFunc,
	}
}

//func init() {
//	wcache.Register("redis", NewRedisCache)
//}
