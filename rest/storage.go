package rest

import (
	"fmt"

	"wgo"
	"wgo/storage"
)

var Storage *storage.Storage

func restStorage() *storage.Storage {
	// 如果rest没有自己的storage, 使用底层wgo的
	if Storage != nil {
		return Storage
	}
	return wgo.Storage()
}

func OpenRedis(cfg *SessionConfig) {
	css := make([]string, 0)
	for _, data := range cfg.Redis {
		css = append(css, fmt.Sprintf("{\"conn\":\"%s\",\"dbNum\":\"%s\"}", data["conn"], data["db"]))
	}
	Storage, _ = storage.New("redis", css...)
}
