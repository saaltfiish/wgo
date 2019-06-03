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
var services map[string]string

func RegisterConfig(tags ...interface{}) {
	if cfg := wgo.SubConfig(tags...); cfg != nil {
		//wgo.Info("find config: %v", cfg)
		config = cfg
		db = config.StringMapString("db")
		es = config.StringMapString("es")
		mio = config.StringMap("storage")
		services = config.StringMapString("services")
	} else {
		Error("not found config for `%s`", tags)
	}
	// open dbs
	if dns := os.Getenv(AECK_DB); dns != "" { // params dns overwrite config file
		OpenDB("db", dns)
	} else if len(db) > 0 {
		for tag, dns := range db {
			OpenDB(tag, dns)
		}
	}

	// es setting
	if len(es) > 0 {
		if ea := os.Getenv(AECK_ES_ADDR); ea != "" { // 可以通过环境变量传入es地址
			es[RCK_ES_ADDR] = ea
		}
		if ri := os.Getenv(AECK_REPORTING_INDEX); ri != "" {
			es[RCK_REPORTING_INDEX] = ri
		}
		if li := os.Getenv(AECK_LOGS_INDEX); li != "" {
			es[RCK_LOGS_INDEX] = li
		}
		Debug("es addr: %s, user: %s, password: %s, indexes: %s, %s", es[RCK_ES_ADDR], es[RCK_ES_USER], es[RCK_ES_PWD], es[RCK_REPORTING_INDEX], es[RCK_LOGS_INDEX])
		if _, ok := es[RCK_REPORTING_INDEX]; !ok {
			es[RCK_REPORTING_INDEX] = "reporting"
		}
		if _, ok := es[RCK_LOGS_INDEX]; !ok {
			es[RCK_LOGS_INDEX] = "asgard-logs"
		}
		OpenElasticSearch()
	}

	// minio
	if len(mio) > 0 {
		Debug("mio endpoint: %s, accessKey: %s, secretKey: %s, cdn_domain: %s, secure: %v", mio["endpoint"].(string), mio["access_key"].(string), mio["secret_key"].(string), mio["cdn"].(string), mio["secure"].(bool))
		openObjectStorage()
	}
}

func SubConfig(key string) *environ.Config {
	return config.Sub(key)
}

// 获取字符串子配置
func ConfigString(key string) string {
	if config != nil {
		return config.String(key)
	}
	return ""
}

// float64
func ConfigFloat(key string) float64 {
	if config != nil {
		return config.Float64(key)
	}
	return 0
}

// string map
func ConfigStringMapString(key string) map[string]string {
	if config != nil {
		return config.StringMapString(key)
	}
	return nil
}

// StringMapStringSlice
func ConfigStringMapStringSlice(key string) map[string][]string {
	if config != nil {
		return config.StringMapStringSlice(key)
	}
	return nil
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

// services, 在rest层管理services
func GetService(key string) string {
	// 优先从env获取, 格式`services.{key}`
	if esa := os.Getenv(fmt.Sprintf("services.%s", key)); esa != "" {
		// env overwrite config
		return esa
	}
	// 如果env没获取, 则从配置文件获取(不建议)
	if key != "" && len(services) > 0 {
		if addr, ok := services[key]; ok {
			return addr
		}
	}
	return ""
}
