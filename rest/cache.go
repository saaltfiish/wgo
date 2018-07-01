package rest

import (
	"encoding/json"
	"fmt"
	"time"

	wcache "wgo/cache"
)

var cache *wcache.Cache

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
	if value = Storage.Get(key); value == nil {
		err = fmt.Errorf("not found %s in redis", key)
	}
	return
}

func RedisSet(key string, value interface{}, expireSeconds int) error {
	var vb []byte
	if vs, ok := value.(string); ok {
		vb = []byte(vs)
	} else {
		vb, _ = json.Marshal(value)
	}
	return Storage.Put(key, vb, time.Duration(expireSeconds)*time.Second)
}

func RedisDel(key string) bool {
	Storage.Delete(key)
	return true
}
