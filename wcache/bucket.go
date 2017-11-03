// Package wcache provides ...
package wcache

import (
	"errors"
	"time"
)

var ErrLargeKey = errors.New("The key is larger than 65535")
var ErrLargeEntry = errors.New("The entry size is larger than 1/1024 of cache size")
var ErrNotFound = errors.New("Entry not found")

type bucket struct {
	id            int
	slots         [256]entry // 每个bucket可装256个对象
	entryCount    int64      // number of entries
	totalCount    int64      // number of entries in bucket, including deleted entries.
	totalTime     int64      // used to calculate least recent used entry.
	totalEvacuate int64      // used for debug
	totalExpired  int64      // used for debug
	overwrites    int64      // used for debug
}

type entry struct {
	key        []byte
	value      interface{}
	accessTime uint32
	expireAt   uint32
	hash16     uint16
	slotId     uint8
}

// new cache
func newBucket(id int) bucket {
	return bucket{
		id: id,
	}
}

// set
func (bkt *bucket) set(key []byte, value interface{}, hashVal uint64, expireSeconds int) (err error) {
	if len(key) > 65535 {
		return ErrLargeKey
	}
	now := uint32(time.Now().Unix())
	expireAt := uint32(0)
	if expireSeconds > 0 {
		expireAt = now + uint32(expireSeconds)
	}

	slotId := uint8(hashVal>>8) >> 4 // range 0 - 15, 每个slot可装16个entry(排序)
	hash16 := uint16(hashVal >> 16)  // range 0 - 65535

	entry := entry{
		key:      key,
		value:    value,
		expireAt: expireAt,
		hash16:   hash16,
		slotId:   slotId,
	}

	start := int32(slotId) * 8
	slot := bkt.slots[start : start+8]
	offset, match := bkt.lookup(slot, hash16, key)
	if match {
		// TODO 同样的key, 可能需要一些
	} else {
		// TODO 根据当前key的热度, 不覆盖当前key, 移到新的地方
	}
	bkt.slots[start+int32(offset)] = entry
	//Error("offset: %d, set entry: %v", offset, bkt.slots[start+int32(offset)])

	return
}

// get
func (bkt *bucket) get(key []byte, hashVal uint64) (value interface{}, err error) {
	if len(key) > 65535 {
		err = ErrLargeKey
		return
	}
	slotId := uint8(hashVal>>8) >> 4 // range 0 - 15, 每个slot可装16个entry(排序)
	hash16 := uint16(hashVal >> 16)  // range 0 - 65535
	start := int32(slotId) * 8
	//Info("slotId: %v, hash16: %v, start: %v", slotId, hash16, start)
	slot := bkt.slots[start : start+8]
	if offset, match := bkt.lookup(slot, hash16, key); match {
		// 查看是否过期
		now := uint32(time.Now().Unix())
		if bkt.slots[start+int32(offset)].expireAt != 0 && bkt.slots[start+int32(offset)].expireAt <= now {
			// TODO, 删除过期的记录
			bkt.totalExpired++
			err = ErrNotFound
			return
		}
		value = bkt.slots[start+int32(offset)].value
	} else {
		err = ErrNotFound
	}
	return
}

// ttl
func (bkt *bucket) ttl(key []byte, hashVal uint64) (timeLeft uint32, err error) {
	return
}

// del
func (bkt *bucket) del(key []byte, hashVal uint64) (affected bool) {
	if len(key) > 65535 {
		return false
	}

	slotId := uint8(hashVal>>8) >> 4 // range 0 - 15, 每个slot可装16个entry(排序)
	hash16 := uint16(hashVal >> 16)  // range 0 - 65535

	start := int32(slotId) * 8
	slot := bkt.slots[start : start+8]
	offset, match := bkt.lookup(slot, hash16, key)
	if match {
		bkt.slots[start+int32(offset)] = entry{}
	}
	return
}

func (bkt *bucket) resetStatistics() {
	bkt.totalEvacuate = 0
	bkt.totalExpired = 0
	bkt.overwrites = 0
}

func (bkt *bucket) lookup(slot []entry, hash16 uint16, key []byte) (idx int, match bool) {
	idx = entryIdx(slot, hash16)
	for idx <= len(slot)-1 { // 可能会有几个hash16相同的排在一起
		ent := slot[idx]
		//Warn("old hash: %d, new: %d", ent.hash16, hash16)
		if ent.hash16 != hash16 {
			break
		}
		if match = ent.key != nil && string(ent.key) == string(key); match {
			return
		}
		if idx < len(slot)-1 {
			idx++
		}
	}
	//Warn("final idx: %v, match: %v", idx, match)
	return
}

// 在一个slot(entry经过排序)范围内, 找到合适的位置
func entryIdx(slot []entry, hash16 uint16) (idx int) {
	high := len(slot) - 1
	for idx < high {
		mid := (idx + high) >> 1
		oldEntry := slot[mid]
		if oldEntry.hash16 < hash16 {
			idx = mid + 1
		} else {
			high = mid
		}
	}
	return
}
