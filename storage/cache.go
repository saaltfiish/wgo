package storage

import (
	"fmt"
	"wgo/storage/core"
	"wgo/storage/redis"
)

// Register makes a cache adapter available by the adapter name.
func Register(name string, adapter core.Instance) {
	if adapter == nil {
		panic("cache: Register adapter is nil")
	}
	if _, ok := core.Adapters[name]; ok {
		panic("cache: Register called twice for adapter " + name)
	}
	core.Adapters[name] = adapter
}

// NewCache Create a new cache driver by adapter name and config string.
func NewCache(adapterName, config string) (adapter core.Cache, err error) {
	instanceFunc, ok := core.Adapters[adapterName]
	if !ok {
		err = fmt.Errorf("cache: unknown adapter name %q (forgot to import?)", adapterName)
		return
	}
	adapter = instanceFunc()
	err = adapter.Start(config)
	if err != nil {
		adapter = nil
	}
	return
}

func init() {
	Register("redis", redis.NewRedisCache)
}
