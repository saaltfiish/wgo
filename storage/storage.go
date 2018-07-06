//
// storage.go
// Copyright (C) 2018 Odin <Odin@Odin-Pro.local>
//
// Distributed under terms of the MIT license.
//

package storage

import (
	"fmt"
	"hash/crc32"
	"time"

	"wgo/storage/core"
)

type Storage struct {
	name  string
	nodes []core.Cache
}

func New(name string, css ...string) (*Storage, error) {
	if name != "" && len(css) > 0 {
		switch name {
		case "redis":
			s := &Storage{
				name:  name,
				nodes: make([]core.Cache, 0),
			}
			for _, cs := range css {
				cache, err := NewCache(name, cs)
				if err != nil {
					panic(err)
				}
				s.nodes = append(s.nodes, cache)
			}
			return s, nil
		default:
			return nil, fmt.Errorf("invalid storage name")
		}
	}
	return nil, fmt.Errorf("storage not created")
}

// Get 根据hash规则查询节点
func (s *Storage) Get(key string) interface{} {
	if key != "" {
		idx := s.Hash(key)
		v := s.nodes[idx].Get(key)
		if v == nil {
			Debug("[Get]idx: %d, key: %s", idx, key)
		}
		return v
	}
	return nil
}

// get and set
func (s *Storage) GetSet(key string, value interface{}) (interface{}, error) {
	if key != "" {
		idx := s.Hash(key)
		return s.nodes[idx].GetSet(key, value)
	}
	return nil, fmt.Errorf("no key")
}

// Put 根据hash规则保存数据
func (s *Storage) Put(key string, val interface{}, timeout time.Duration, opts ...interface{}) error {
	idx := s.Hash(key)
	//go s.nodes[idx].Put(key, val, timeout)
	tried := 0
	for tried < 5 {
		tried++
		if err := s.nodes[idx].Put(key, val, timeout, opts...); err == nil {
			break
		}
	}
	Debug("[Put]idx: %d, key: %s, tried: %d", idx, key, tried)
	return nil
}

// Delete 根据hash规则删除数据
func (s *Storage) Delete(key string) error {
	idx := s.Hash(key)
	go s.nodes[idx].Delete(key)
	return nil
}

func (s *Storage) Hash(key string) int {
	if key == "" {
		panic("keys error")
	}
	sub := uint(crc32.ChecksumIEEE([]byte(key)) % 1024)
	//return int(sub) / (1024 / len(s.nodes))
	return int(float64(sub) / (float64(1024) / float64(len(s.nodes))))
}
