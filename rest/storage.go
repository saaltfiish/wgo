package rest

import (
	"fmt"
	"hash/crc32"
	"time"

	"gitlab.intra.wepiao.com/arch/wcache"
	"gitlab.intra.wepiao.com/arch/wcache/core"
)

type storage struct {
	name  string
	nodes []core.Cache
}

var Storage *storage

// Start 顾名思义
func OpenRedis(cfg *SessionConfig) {
	Storage = &storage{
		name:  "redis",
		nodes: make([]core.Cache, 0),
	}
	for _, data := range cfg.Redis {
		confStr := fmt.Sprintf("{\"prefix\":\"%s\",\"conn\":\"%s\",\"dbNum\":\"%s\"}", cfg.Prefix, data["conn"], data["db"])
		//Info("conf: %s", confStr)
		cache, err := wcache.NewCache(Storage.name, confStr)
		if err != nil {
			panic(err)
		}
		Storage.nodes = append(Storage.nodes, cache)
	}
}

// Get 根据hash规则查询节点
func (s *storage) Get(key string) interface{} {
	idx := s.Hash(key)
	v := s.nodes[idx].Get(key)
	if v == nil {
		Info("[Get]idx: %d, key: %s", idx, key)
	}
	return v
}

// Put 根据hash规则保存数据
func (s *storage) Put(key string, val interface{}, timeout time.Duration) error {
	idx := s.Hash(key)
	//go s.nodes[idx].Put(key, val, timeout)
	tried := 0
	for tried < 5 {
		tried++
		if err := s.nodes[idx].Put(key, val, timeout); err == nil {
			break
		}
	}
	Info("[Put]idx: %d, key: %s, tried: %d", idx, key, tried)
	return nil
}

// Delete 根据hash规则删除数据
func (s *storage) Delete(key string) error {
	idx := s.Hash(key)
	go s.nodes[idx].Delete(key)
	return nil
}

func (s *storage) Hash(key string) int {
	if key == "" {
		panic("keys error")
	}
	sub := uint(crc32.ChecksumIEEE([]byte(key)) % 1024)
	//return int(sub) / (1024 / len(s.nodes))
	return int(float64(sub) / (float64(1024) / float64(len(s.nodes))))
}
