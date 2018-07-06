//
// storage.go
// Copyright (C) 2018 Odin <Odin@Odin-Pro.local>
//
// Distributed under terms of the MIT license.
//

package wgo

import (
	"fmt"
	"os"
	"strings"

	"wgo/environ"
	"wgo/storage"
)

// init storage
func initStorage() {
	var sn string
	var nodes []string
	if rns := os.Getenv("storage.redis.nodes"); rns != "" {
		sn = "redis"
		nodes = strings.Split(rns, ",")
	} else if scfg := SubConfig(environ.CFG_KEY_STORAGE); scfg != nil {
		// nodes可以通过env传递
		sn = scfg.String("name")
		nodes = scfg.StringSlice("nodes")
	}
	if sn != "" && len(nodes) > 0 {
		// 配置了storage再进行初始化
		css := make([]string, 0)
		for _, node := range nodes {
			Info("open %s storage, %s", sn, node)
			split := strings.Split(node, ":")
			if len(split) > 0 && split[0] != "" {
				host := split[0]
				port := "6379"
				db := "0"
				if len(split) >= 2 && split[1] != "" {
					port = split[1]
				}
				if len(split) >= 3 && split[2] != "" {
					db = split[2]
				}
				css = append(css, fmt.Sprintf("{\"conn\":\"%s:%s\",\"dbNum\":\"%s\"}", host, port, db))
			}
		}
		s, err := storage.New(sn, css...)
		if err != nil {
			Fatal("open storage failed: %s", err)
		}
		SetStorage(s)
	}
}

// storage
// get
func Storage() *storage.Storage { return wgo.Storage() }
func (w *WGO) Storage() *storage.Storage {
	return w.storage
}

// set storage
func SetStorage(s *storage.Storage) {
	wgo.SetStorage(s)
}
func (w *WGO) SetStorage(s *storage.Storage) {
	if w != nil {
		w.storage = s
	}
}
