package core

import (
	"time"
)

type Cache interface {
	Get(key string) interface{}
	HGet(key string, filed string) interface{}
	GetMulti(keys []string) []interface{}
	Put(key string, val interface{}, timeout time.Duration) error
	Delete(key string) error
	Incr(key string) (int, error)
	Decr(key string) (int, error)
	IncrBy(key string, num int) (int, error)
	DecrBy(key string, num int) (int, error)
	Push(key string, val interface{}) error
	Pop(key string) (interface{}, error)
	IsExist(key string) bool
	ClearAll() error
	Start(config string) error
}

// Instance is a function create a new Cache Instance
type Instance func() Cache

var Adapters = make(map[string]Instance)
