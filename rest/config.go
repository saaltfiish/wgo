// Package rest provides ...
package rest

import (
	"encoding/json"
	"fmt"
	"os"

	yaml "gopkg.in/yaml.v2"

	"wgo"
	"wgo/environ"
)

// type Config struct {
// 	DB map[string]string `json:"db"`
// 	ES map[string]string `json:"es"`
// }

var config *environ.Config
var db map[string]string
var es map[string]string
var mio map[string]interface{} // object storage

func RegisterConfig(tags ...interface{}) {
	if cfg := wgo.SubConfig(tags...); cfg != nil {
		//wgo.Info("find config: %v", cfg)
		config = cfg
		db = config.StringMapString("db")
		es = config.StringMapString("es")
		mio = config.StringMap("storage")
	} else {
		panic("not found config")
	}
	// open dbs
	if dns := os.Getenv("rest.db"); dns != "" { // params dns overwrite config file
		OpenDB("db", dns)
	} else if len(db) > 0 {
		for tag, dns := range db {
			OpenDB(tag, dns)
		}
	}

	// es setting
	if len(es) > 0 {
		if ea := os.Getenv("rest.esaddr"); ea != "" { // 可以通过环境变量传入es地址
			es["addr"] = ea
		}
		Debug("es addr: %s, index: %s, user: %s, password: %s", es["addr"], es["index"], es["user"], es["password"])
		OpenElasticSearch()
	}

	// minio
	if len(mio) > 0 {
		Debug("mio endpoint: %s, accessKey: %s, secretKey: %s, secure: %v", mio["endpoint"].(string), mio["access_key"].(string), mio["secret_key"].(string), mio["secure"].(bool))
		openObjectStorage()
	}
}

// 获取深层config
func GetConfig(rawVal interface{}, opts ...interface{}) error {
	// default key app
	if len(opts) > 0 && config != nil {
		if k, ok := opts[0].(string); ok {
			if environ.ConfigType() == "yaml" {
				bytes, _ := yaml.Marshal(config.Get(k))
				// Debug("string: %s", string(jsonBytes))
				return yaml.Unmarshal(bytes, rawVal)
			} else {
				bytes, _ := json.Marshal(config.Get(k))
				// Debug("string%s", string(jsonBytes))
				return json.Unmarshal(bytes, rawVal)
			}
		}
	}
	return fmt.Errorf("not found config for %s", opts)
}
