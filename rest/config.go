// Package rest provides ...
package rest

import (
	"os"

	"wgo"
)

var config *Config

type Config struct {
	DB map[string]string `json:"db"`
	ES map[string]string `json:"es"`
}

func RegisterConfig(tags ...interface{}) {
	cfg := new(Config)
	if err := wgo.AppConfig(cfg, tags...); err == nil {
		//wgo.Info("find config: %v", cfg)
		config = cfg
	}
	// open dbs
	if dns := os.Getenv("rest.db"); dns != "" { // params dns overwrite config file
		OpenDB("db", dns)
	} else if len(config.DB) > 0 {
		for tag, dns := range config.DB {
			OpenDB(tag, dns)
		}
	}

	// es setting
	if len(config.ES) > 0 {
		if ea := os.Getenv("rest.esaddr"); ea != "" { // 可以通过环境变量传入es地址
			config.ES["addr"] = ea
		}
		Info("es addr: %s, index: %s, user: %s, password: %s", config.ES["addr"], config.ES["index"], config.ES["user"], config.ES["password"])
		OpenElasticSearch()
	}
}
