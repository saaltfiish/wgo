package rest

import (
	"encoding/json"
	"fmt"
	"time"

	wcache "wgo/cache"
)

var cache *wcache.Cache = wcache.NewCache()

func LocalGet(key string) (value interface{}, err error) {
	if key != "" {
		return cache.Get([]byte(key))
	}
	return
}

func LocalSet(key string, value interface{}, expireSeconds int) error {
	return cache.Set([]byte(key), value, expireSeconds)
}

func LocalDel(key string) bool {
	return cache.Del([]byte(key))
}

func RedisGet(key string) (value interface{}, err error) {
	if s := restStorage(); s != nil {
		if value = s.Get(key); value == nil {
			err = fmt.Errorf("not found %s in redis", key)
		}
	} else {
		err = fmt.Errorf("not found storage")
	}
	return
}

func RedisSet(key string, value interface{}, expireSeconds int) error {
	var vb []byte
	if _, ok := value.([]byte); ok {
		vb = value.([]byte)
	} else if vs, ok := value.(string); ok {
		vb = []byte(vs)
	} else {
		vb, _ = json.Marshal(value)
	}
	if s := restStorage(); s != nil {
		return s.Put(key, vb, time.Duration(expireSeconds)*time.Second)
	} else {
		return fmt.Errorf("not found storage")
	}
}

func RedisDel(key string) bool {
	if s := restStorage(); s != nil {
		s.Delete(key)
		return true
	} else {
		return false
	}
}
