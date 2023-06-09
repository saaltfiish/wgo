package utils

import (
	"sync"
)

type SafeMap struct {
	lock *sync.RWMutex
	bm   map[interface{}]interface{}
}

// NewSafeMap return new safemap
// 可传入一个存在的map作为初始map
func NewSafeMap(i ...map[interface{}]interface{}) *SafeMap {
	var bm map[interface{}]interface{}
	if len(i) > 0 {
		bm = i[0]
	} else {
		bm = make(map[interface{}]interface{})
	}
	return &SafeMap{
		lock: new(sync.RWMutex),
		bm:   bm,
	}
}

// Get from maps return the k's value
func (m *SafeMap) Get(k interface{}) interface{} {
	m.lock.RLock()
	defer m.lock.RUnlock()
	if val, ok := m.bm[k]; ok {
		return val
	}
	return nil
}

// Maps the given key and value. Returns false
// if the key is already in the map and changes nothing.
func (m *SafeMap) Set(k interface{}, v interface{}) bool {
	m.lock.Lock()
	defer m.lock.Unlock()
	if val, ok := m.bm[k]; !ok {
		m.bm[k] = v
	} else if val != v {
		m.bm[k] = v
	} else {
		return false
	}
	return true
}

// Returns true if k is exist in the map.
func (m *SafeMap) Check(k interface{}) bool {
	m.lock.RLock()
	defer m.lock.RUnlock()
	if _, ok := m.bm[k]; !ok {
		return false
	}
	return true
}

// Delete the given key and value.
func (m *SafeMap) Delete(k interface{}) {
	m.lock.Lock()
	defer m.lock.Unlock()
	delete(m.bm, k)
}

// Items returns all items in safemap.
func (m *SafeMap) Items() map[interface{}]interface{} {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return m.bm
}

// Clone returns a copied safemap
func (m *SafeMap) Clone() *SafeMap {
	nm := NewSafeMap()
	for k, v := range m.Items() {
		nm.Set(k, v)
	}
	return nm
}
