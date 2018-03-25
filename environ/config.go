package environ

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	"github.com/spf13/viper"
	"wgo/utils"
)

const (
	CFG_KEY_PROCNAME    = "proc_name"
	CFG_KEY_DOCKERIZE   = "dockerize"
	CFG_KEY_ENABLECACHE = "enable_cache"
	CFG_KEY_DAEMONIZE   = "daemonize"
	CFG_KEY_DEBUG       = "debug"
	CFG_KEY_APPDIR      = "app_dir"
	CFG_KEY_WORKDIR     = "work_dir"
	CFG_KEY_CONFDIR     = "conf_dir"
	CFG_KEY_PIDFILE     = "pid_file"
	CFG_KEY_CONFFILE    = "conf_file"
	CFG_KEY_TIMEZONE    = "time_zone"
	CFG_KEY_LOGS        = "logs"
	CFG_KEY_SERVERS     = "servers"
	CFG_KEY_ENGINE      = "engine"
	CFG_KEY_MODE        = "mode"
	CFG_KEY_LISTEN      = "listen"
	CFG_KEY_ADDR        = "addr"
	CFG_KEY_PORT        = "port"
	CFG_KEY_HOSTS       = "hosts"
	CFG_KEY_VERSION     = "version"
	CFG_KEY_ACCESS      = "access"
)

type (
	Config struct {
		v *viper.Viper
	}
)

/* {{{ func NewConfig(ops ...interface{}) *Config
 *
 */
func NewConfig(ops ...interface{}) *Config {
	if len(ops) > 0 {
		if v, ok := ops[0].(*viper.Viper); ok {
			return &Config{v: v}
		}
	}
	return &Config{v: viper.New()}
}

/* }}} */

/* {{{ func DefaultConfig() *Config
 *
 */
func DefaultConfig() *Config {
	cfg := NewConfig()

	// 优先命令行或者环境变量的配置文件
	if cf, ct := getConfigInfoFromFlags(); cf != "" && ct != "" { // 命令行
		cfg.ReadInConfig(cf, ct)
	} else if cf, ct := getConfigInfoFromSysEnv(); cf != "" && ct != "" { // 系统环境变量
		cfg.ReadInConfig(cf, ct)
	} else if executeName != "" { // 先在app目录下conf目录找, 其次是app目录, 第三是当前目录, 默认以执行文件为名, 后缀可为"json,yaml,yml等"
		cfg.v.SetConfigName(executeName)
		cfg.v.AddConfigPath(filepath.Join(executeDir, "conf"))
		cfg.v.AddConfigPath(defaultWorkDir)
		cfg.v.AddConfigPath(".")
		if err := cfg.v.ReadInConfig(); err != nil {
			panic(fmt.Sprintf("[PANIC] read config file failed: %s", err))
		}
	}

	return cfg
}

/* }}} */

/* {{{ func (cfg *Config) ReadInConfig() *Config
 * 从字符串获取配置
 */
func (cfg *Config) ReadConfig(cc, ct string) *Config {

	if cc != "" && ct != "" {
		cfg.v.SetConfigType(ct)
		ccb := []byte(cc)
		if err := cfg.v.ReadConfig(bytes.NewBuffer(ccb)); err != nil {
			panic(fmt.Sprintf("[PANIC] read config string failed: %s", err))
		}
	}

	return cfg
}

/* }}} */

/* {{{ func (cfg *Config) ReadInConfig() *Config
 * 从文件获取配置
 */
func (cfg *Config) ReadInConfig(cf, ct string) *Config {

	if cf != "" && ct != "" && utils.FileExists(cf) {
		cfg.v.Set(CFG_KEY_CONFFILE, cf)
		cfg.v.SetConfigFile(cf)
		cfg.v.SetConfigType(ct)
		if err := cfg.v.ReadInConfig(); err != nil {
			panic(fmt.Sprintf("[PANIC] read config file failed: %s", err))
		}
	}

	return cfg
}

/* }}} */

/* {{{ func getConfigInfoFromFlags() (string, string)
 *
 */
func getConfigInfoFromFlags() (string, string) {
	//var cf, ct string
	cf := StringFlag(FLAG_KEY_CONFIGFILE)
	if cf != "" && !utils.FileExists(cf) { //文件不存在
		return "", ""
	}
	return cf, StringFlag(FLAG_KEY_CONFIGTYPE)
}

/* }}} */

/* {{{ func getConfigInfoFromSysEnv() (string, string)
 * TODO
 */
func getConfigInfoFromSysEnv() (string, string) {
	return "", ""
}

/* }}} */

/* {{{ func (cfg *Config) Get(key string) interface{}
 * 封装viper方法
 */
func (cfg *Config) Get(key string) interface{} {
	return cfg.v.Get(key)
}

/* }}} */

/* {{{ func (cfg *Config) String(key string) string
 * 封装viper方法
 */
func (cfg *Config) String(key string) string {
	return cfg.v.GetString(key)
}

/* }}} */

/* {{{ func (cfg *Config) Bool(key string) bool
 *
 */
func (cfg *Config) Bool(key string) bool {
	return cfg.v.GetBool(key)
}

/* }}} */

/* {{{ func (cfg *Config) Int(key string) int
 *
 */
func (cfg *Config) Int(key string) int {
	return cfg.v.GetInt(key)
}

/* }}} */

/* {{{ func (cfg *Config) Float64(key string) float64
 *
 */
func (cfg *Config) Float64(key string) float64 {
	return cfg.v.GetFloat64(key)
}

/* }}} */

/* {{{ func (cfg *Config) Time(key string) time.Time
 *
 */
func (cfg *Config) Time(key string) time.Time {
	return cfg.v.GetTime(key)
}

/* }}} */

/* {{{ func (cfg *Config) Duration(key string) time.Duration
 *
 */
func (cfg *Config) Duration(key string) time.Duration {
	return cfg.v.GetDuration(key)
}

/* }}} */

/* {{{ func (cfg *Config) StringSlice(key string) []string
 *
 */
func (cfg *Config) StringSlice(key string) []string {
	return cfg.v.GetStringSlice(key)
}

/* }}} */

/* {{{ func (cfg *Config) StringMap(key string) map[string]interface{}
 *
 */
func (cfg *Config) StringMap(key string) map[string]interface{} {
	return cfg.v.GetStringMap(key)
}

/* }}} */

/* {{{ func (cfg *Config) StringMapString(key string) map[string]string
 *
 */
func (cfg *Config) StringMapString(key string) map[string]string {
	return cfg.v.GetStringMapString(key)
}

/* }}} */

/* {{{ func (cfg *Config) StringMapStringSlice(key string) map[string][]string
 *
 */
func (cfg *Config) StringMapStringSlice(key string) map[string][]string {
	return cfg.v.GetStringMapStringSlice(key)
}

/* }}} */

/* {{{ func (cfg *Config) IsSet(key string) bool
 *
 */
func (cfg *Config) IsSet(key string) bool {
	return cfg.v.IsSet(key)
}

/* }}} */

/* {{{ func (cfg *Config) Sub(key string) *Config
 *
 */
func (cfg *Config) Sub(key string) *Config {
	return NewConfig(cfg.v.Sub(key))
}

/* }}} */

/* {{{ func (cfg *Config) UnmarshalKey(key string, rawVal interface{}) error
 *
 */
func (cfg *Config) UnmarshalKey(key string, rawVal interface{}) error {
	return cfg.v.UnmarshalKey(key, rawVal)
}

/* }}} */

/* {{{ func (cfg *Config) AppConfig(rawVal interface{}, opts ...interface{}) interface{}
 * 封装viper方法
 */
func (cfg *Config) AppConfig(rawVal interface{}, opts ...interface{}) error {
	key := "app"
	if len(opts) > 0 {
		if k, ok := opts[0].(string); ok {
			key = k
		}
	}
	jsonBytes, _ := json.Marshal(cfg.v.Get(key))
	return json.Unmarshal(jsonBytes, rawVal)

	//return cfg.v.UnmarshalKey(key, rawVal) // 跟系统json包反序列化有区别
}

/* }}} */
