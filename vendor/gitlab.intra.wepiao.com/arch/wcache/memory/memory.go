package memory

import (
	"encoding/json"
	"errors"
	"gitlab.intra.wepiao.com/arch/wcache/core"
	"sync"
	"time"
)

var (
	// DefaultEvery means the clock time of recycling the expired cache items in memory.
	DefaultEvery = 60 // 1 minute
)

// MemoryItem store memory cache item.
type MemoryItem struct {
	val         interface{}
	createdTime time.Time
	lifespan    time.Duration
}

func (mi *MemoryItem) isExpire() bool {
	// 0 means forever
	if mi.lifespan == 0 {
		return false
	}
	return time.Now().Sub(mi.createdTime) > mi.lifespan
}

// MemoryCache is Memory cache adapter.
// it contains a RW locker for safe map storage.
type Cache struct {
	sync.RWMutex
	dur   time.Duration
	items map[string]*MemoryItem
	Every int // run an expiration check Every clock time
}

// NewMemoryCache returns a new MemoryCache.
func NewMemoryCache() core.Cache {
	cache := Cache{items: make(map[string]*MemoryItem)}
	return &cache
}

// Get cache from memory.
// if non-existed or expired, return nil.
func (bc *Cache) Get(name string) interface{} {
	bc.RLock()
	defer bc.RUnlock()
	if itm, ok := bc.items[name]; ok {
		if itm.isExpire() {
			return nil
		}
		return itm.val
	}
	return nil
}

// Get cache from redis.
func (rc *Cache) HGet(key string, filed string) interface{} {
	return errors.New("memory no")
}

// GetMulti gets caches from memory.
// if non-existed or expired, return nil.
func (bc *Cache) GetMulti(names []string) []interface{} {
	var rc []interface{}
	for _, name := range names {
		rc = append(rc, bc.Get(name))
	}
	return rc
}

// Put cache to memory.
// if lifespan is 0, it will be forever till restart.
func (bc *Cache) Put(name string, value interface{}, lifespan time.Duration) error {
	bc.Lock()
	defer bc.Unlock()
	bc.items[name] = &MemoryItem{
		val:         value,
		createdTime: time.Now(),
		lifespan:    lifespan,
	}
	return nil
}

// Delete cache in memory.
func (bc *Cache) Delete(name string) error {
	bc.Lock()
	defer bc.Unlock()
	if _, ok := bc.items[name]; !ok {
		return errors.New("key not exist")
	}
	delete(bc.items, name)
	if _, ok := bc.items[name]; ok {
		return errors.New("delete key error")
	}
	return nil
}

// Incr increase cache counter in memory.
// it supports int,int32,int64,uint,uint32,uint64.
func (bc *Cache) Incr(key string) (int, error) {
	bc.RLock()
	defer bc.RUnlock()
	itm, ok := bc.items[key]
	if !ok {
		return 0, errors.New("key not exist")
	}
	switch itm.val.(type) {
	case int:
		itm.val = itm.val.(int) + 1
	case int32:
		itm.val = itm.val.(int32) + 1
	case int64:
		itm.val = itm.val.(int64) + 1
	case uint:
		itm.val = itm.val.(uint) + 1
	case uint32:
		itm.val = itm.val.(uint32) + 1
	case uint64:
		itm.val = itm.val.(uint64) + 1
	default:
		return 0, errors.New("item val is not (u)int (u)int32 (u)int64")
	}
	return int(itm.val.(uint64)), nil
}

// Decr decrease counter in memory.
func (bc *Cache) Decr(key string) (int, error) {
	bc.RLock()
	defer bc.RUnlock()
	itm, ok := bc.items[key]
	if !ok {
		return 0, errors.New("key not exist")
	}
	switch itm.val.(type) {
	case int:
		itm.val = itm.val.(int) - 1
	case int64:
		itm.val = itm.val.(int64) - 1
	case int32:
		itm.val = itm.val.(int32) - 1
	case uint:
		if itm.val.(uint) > 0 {
			itm.val = itm.val.(uint) - 1
		} else {
			return 0, errors.New("item val is less than 0")
		}
	case uint32:
		if itm.val.(uint32) > 0 {
			itm.val = itm.val.(uint32) - 1
		} else {
			return 0, errors.New("item val is less than 0")
		}
	case uint64:
		if itm.val.(uint64) > 0 {
			itm.val = itm.val.(uint64) - 1
		} else {
			return 0, errors.New("item val is less than 0")
		}
	default:
		return 0, errors.New("item val is not int int64 int32")
	}
	return int(itm.val.(uint64)), nil
}

// IncrBy increase cache counter in memory.
// it supports int,int32,int64,uint,uint32,uint64.
func (bc *Cache) IncrBy(key string, num int) (int, error) {
	bc.RLock()
	defer bc.RUnlock()
	itm, ok := bc.items[key]
	if !ok {
		return 0, errors.New("key not exist")
	}
	switch itm.val.(type) {
	case int:
		itm.val = itm.val.(int) + num
	case int32:
		itm.val = itm.val.(int32) + int32(num)
	case int64:
		itm.val = itm.val.(int64) + int64(num)
	case uint:
		itm.val = itm.val.(uint) + uint(num)
	case uint32:
		itm.val = itm.val.(uint32) + uint32(num)
	case uint64:
		itm.val = itm.val.(uint64) + uint64(num)
	default:
		return 0, errors.New("item val is not (u)int (u)int32 (u)int64")
	}
	return int(itm.val.(uint64)), nil
}

// DecrBy decrease counter in memory.
func (bc *Cache) DecrBy(key string, num int) (int, error) {
	bc.RLock()
	defer bc.RUnlock()
	itm, ok := bc.items[key]
	if !ok {
		return 0, errors.New("key not exist")
	}
	switch itm.val.(type) {
	case int:
		itm.val = itm.val.(int) - num
	case int64:
		itm.val = itm.val.(int64) - int64(num)
	case int32:
		itm.val = itm.val.(int32) - int32(num)
	case uint:
		if itm.val.(uint) > 0 {
			itm.val = itm.val.(uint) - uint(num)
		} else {
			return 0, errors.New("item val is less than 0")
		}
	case uint32:
		if itm.val.(uint32) > 0 {
			itm.val = itm.val.(uint32) - uint32(num)
		} else {
			return 0, errors.New("item val is less than 0")
		}
	case uint64:
		if itm.val.(uint64) > 0 {
			itm.val = itm.val.(uint64) - uint64(num)
		} else {
			return 0, errors.New("item val is less than 0")
		}
	default:
		return 0, errors.New("item val is not int int64 int32")
	}
	return int(itm.val.(uint64)), nil
}

// Push push
func (fc *Cache) Push(key string, val interface{}) error {
	return errors.New("file no")
}

// Pop pop
func (fc *Cache) Pop(key string) (interface{}, error) {
	return nil, errors.New("file no")
}

// IsExist check cache exist in memory.
func (bc *Cache) IsExist(name string) bool {
	bc.RLock()
	defer bc.RUnlock()
	if v, ok := bc.items[name]; ok {
		return !v.isExpire()
	}
	return false
}

// ClearAll will delete all cache in memory.
func (bc *Cache) ClearAll() error {
	bc.Lock()
	defer bc.Unlock()
	bc.items = make(map[string]*MemoryItem)
	return nil
}

// Start start memory cache. it will check expiration in every clock time.
func (bc *Cache) Start(config string) error {
	var cf map[string]int
	json.Unmarshal([]byte(config), &cf)
	if _, ok := cf["interval"]; !ok {
		cf = make(map[string]int)
		cf["interval"] = DefaultEvery
	}
	dur := time.Duration(cf["interval"]) * time.Second
	bc.Every = cf["interval"]
	bc.dur = dur
	go bc.vaccuum()
	return nil
}

// check expiration.
func (bc *Cache) vaccuum() {
	if bc.Every < 1 {
		return
	}
	for {
		<-time.After(bc.dur)
		if bc.items == nil {
			return
		}
		for name := range bc.items {
			bc.itemExpired(name)
		}
	}
}

// itemExpired returns true if an item is expired.
func (bc *Cache) itemExpired(name string) bool {
	bc.Lock()
	defer bc.Unlock()

	itm, ok := bc.items[name]
	if !ok {
		return true
	}
	if itm.isExpire() {
		delete(bc.items, name)
		return true
	}
	return false
}

//func init() {
//	wcache.Register("memory", NewMemoryCache)
//}
