package memcache

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/bradfitz/gomemcache/memcache"

	"time"

	"gitlab.intra.wepiao.com/arch/wcache/core"
)

// Cache Memcache adapter.
type Cache struct {
	conn     *memcache.Client
	conninfo []string
}

// NewMemCache create new memcache adapter.
func NewMemCache() core.Cache {
	return &Cache{}
}

// Get get value from memcache.
func (rc *Cache) Get(key string) interface{} {
	if rc.conn == nil {
		if err := rc.connectInit(); err != nil {
			return err
		}
	}
	if item, err := rc.conn.Get(key); err == nil {
		return string(item.Value)
	}
	return nil
}

// Get cache from redis.
func (rc *Cache) HGet(key string, filed string) interface{} {
	return errors.New("memcache no")
}

// GetMulti get value from memcache.
func (rc *Cache) GetMulti(keys []string) []interface{} {
	size := len(keys)
	var rv []interface{}
	if rc.conn == nil {
		if err := rc.connectInit(); err != nil {
			for i := 0; i < size; i++ {
				rv = append(rv, err)
			}
			return rv
		}
	}
	mv, err := rc.conn.GetMulti(keys)
	if err == nil {
		for _, v := range mv {
			rv = append(rv, string(v.Value))
		}
		return rv
	}
	for i := 0; i < size; i++ {
		rv = append(rv, err)
	}
	return rv
}

// Put put value to memcache. only support string.
func (rc *Cache) Put(key string, val interface{}, timeout time.Duration) error {
	if rc.conn == nil {
		if err := rc.connectInit(); err != nil {
			return err
		}
	}
	v, ok := val.(string)
	if !ok {
		return errors.New("val must string")
	}
	item := memcache.Item{Key: key, Value: []byte(v), Expiration: int32(timeout / time.Second)}
	return rc.conn.Set(&item)
}

// Delete delete value in memcache.
func (rc *Cache) Delete(key string) error {
	if rc.conn == nil {
		if err := rc.connectInit(); err != nil {
			return err
		}
	}
	return rc.conn.Delete(key)
}

// Incr increase counter.
func (rc *Cache) Incr(key string) (int, error) {
	if rc.conn == nil {
		if err := rc.connectInit(); err != nil {
			return 0, err
		}
	}
	data, err := rc.conn.Increment(key, 1)
	return int(data), err
}

// Decr decrease counter.
func (rc *Cache) Decr(key string) (int, error) {
	if rc.conn == nil {
		if err := rc.connectInit(); err != nil {
			return 0, err
		}
	}
	data, err := rc.conn.Decrement(key, 1)
	return int(data), err
}

// IncrBy increase counter.
func (rc *Cache) IncrBy(key string, num int) (int, error) {
	if rc.conn == nil {
		if err := rc.connectInit(); err != nil {
			return 0, err
		}
	}
	data, err := rc.conn.Increment(key, uint64(num))
	return int(data), err
}

// DecrBy decrease counter.
func (rc *Cache) DecrBy(key string, num int) (int, error) {
	if rc.conn == nil {
		if err := rc.connectInit(); err != nil {
			return 0, err
		}
	}
	data, err := rc.conn.Decrement(key, uint64(num))
	return int(data), err
}

// Push push
func (rc *Cache) Push(key string, val interface{}) error {
	return errors.New("file no")
}

// Pop pop
func (rc *Cache) Pop(key string) (interface{}, error) {
	return nil, errors.New("file no")
}

// IsExist check value exists in memcache.
func (rc *Cache) IsExist(key string) bool {
	if rc.conn == nil {
		if err := rc.connectInit(); err != nil {
			return false
		}
	}
	_, err := rc.conn.Get(key)
	if err != nil {
		return false
	}
	return true
}

// ClearAll clear all cached in memcache.
func (rc *Cache) ClearAll() error {
	if rc.conn == nil {
		if err := rc.connectInit(); err != nil {
			return err
		}
	}
	return rc.conn.FlushAll()
}

// Start start memcache adapter.
// config string is like {"conn":"connection info"}.
// if connecting error, return.
func (rc *Cache) Start(config string) error {
	var cf map[string]string
	json.Unmarshal([]byte(config), &cf)
	if _, ok := cf["conn"]; !ok {
		return errors.New("config has no conn key")
	}
	rc.conninfo = strings.Split(cf["conn"], ";")
	if rc.conn == nil {
		if err := rc.connectInit(); err != nil {
			return err
		}
	}
	return nil
}

// connect to memcache and keep the connection.
func (rc *Cache) connectInit() error {
	rc.conn = memcache.New(rc.conninfo...)
	return nil
}

//func init() {
//	wcache.Register("memcache", NewMemCache)
//}
