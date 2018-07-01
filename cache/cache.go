// Package wcache provides ...
package cache

import (
	"sync"
	"sync/atomic"

	"github.com/spaolacci/murmur3"
)

type Cache struct {
	locks     [256]sync.Mutex
	buckets   [256]bucket // 一共分256桶
	hitCount  int64
	missCount int64
}

func hashFunc(data []byte) uint64 {
	return murmur3.Sum64(data)
}

// The cache size will be set to 512KB at minimum.
// If the size is set relatively large, you should call
// `debug.SetGCPercent()`, set it to a much smaller value
// to limit the memory consumption and GC pause time.
func NewCache() (cache *Cache) {
	cache = new(Cache)
	for i := 0; i < 256; i++ {
		cache.buckets[i] = newBucket(i)
	}
	return
}

// cache set
func (cache *Cache) Set(key []byte, value interface{}, expireSeconds int) (err error) {
	hashVal := hashFunc(key)
	bktId := hashVal & 255
	//Info("hashVal: %v, bucket: %v", hashVal, bktId)
	cache.locks[bktId].Lock()
	err = cache.buckets[bktId].set(key, value, hashVal, expireSeconds)
	cache.locks[bktId].Unlock()
	return
}

// Get the value or not found error.
func (cache *Cache) Get(key []byte) (value interface{}, err error) {
	hashVal := hashFunc(key)
	bktId := hashVal & 255
	//Info("hashVal: %v, bucket: %v", hashVal, bktId)
	cache.locks[bktId].Lock()
	value, err = cache.buckets[bktId].get(key, hashVal)
	cache.locks[bktId].Unlock()
	if err == nil {
		atomic.AddInt64(&cache.hitCount, 1)
	} else {
		atomic.AddInt64(&cache.missCount, 1)
	}
	return
}

func (cache *Cache) TTL(key []byte) (timeLeft uint32, err error) {
	hashVal := hashFunc(key)
	bktId := hashVal & 255
	timeLeft, err = cache.buckets[bktId].ttl(key, hashVal)
	return
}

func (cache *Cache) Del(key []byte) (affected bool) {
	hashVal := hashFunc(key)
	bktId := hashVal & 255
	cache.locks[bktId].Lock()
	affected = cache.buckets[bktId].del(key, hashVal)
	cache.locks[bktId].Unlock()
	return
}

// 撤回次数
func (cache *Cache) EvacuateCount() (count int64) {
	for i := 0; i < 256; i++ {
		count += atomic.LoadInt64(&cache.buckets[i].totalEvacuate)
	}
	return
}

// 过期次数
func (cache *Cache) ExpiredCount() (count int64) {
	for i := 0; i < 256; i++ {
		count += atomic.LoadInt64(&cache.buckets[i].totalExpired)
	}
	return
}

// 记录数
func (cache *Cache) EntryCount() (entryCount int64) {
	for i := 0; i < 256; i++ {
		entryCount += atomic.LoadInt64(&cache.buckets[i].entryCount)
	}
	return
}

// The average unix timestamp when a entry being accessed.
// Entries have greater access time will be evacuated when it
// is about to be overwritten by new value.
func (cache *Cache) AverageAccessTime() int64 {
	var entryCount, totalTime int64
	for i := 0; i < 256; i++ {
		totalTime += atomic.LoadInt64(&cache.buckets[i].totalTime)
		entryCount += atomic.LoadInt64(&cache.buckets[i].totalCount)
	}
	if entryCount == 0 {
		return 0
	} else {
		return totalTime / entryCount
	}
}

func (cache *Cache) HitCount() int64 {
	return atomic.LoadInt64(&cache.hitCount)
}

func (cache *Cache) LookupCount() int64 {
	return atomic.LoadInt64(&cache.hitCount) + atomic.LoadInt64(&cache.missCount)
}

func (cache *Cache) HitRate() float64 {
	lookupCount := cache.LookupCount()
	if lookupCount == 0 {
		return 0
	} else {
		return float64(cache.HitCount()) / float64(lookupCount)
	}
}

func (cache *Cache) OverwriteCount() (overwriteCount int64) {
	for i := 0; i < 256; i++ {
		overwriteCount += atomic.LoadInt64(&cache.buckets[i].overwrites)
	}
	return
}

func (cache *Cache) Clear() {
	for i := 0; i < 256; i++ {
		cache.locks[i].Lock()
		newBkt := newBucket(i)
		cache.buckets[i] = newBkt
		cache.locks[i].Unlock()
	}
	atomic.StoreInt64(&cache.hitCount, 0)
	atomic.StoreInt64(&cache.missCount, 0)
}

func (cache *Cache) ResetStatistics() {
	atomic.StoreInt64(&cache.hitCount, 0)
	atomic.StoreInt64(&cache.missCount, 0)
	for i := 0; i < 256; i++ {
		cache.locks[i].Lock()
		cache.buckets[i].resetStatistics()
		cache.locks[i].Unlock()
	}
}
