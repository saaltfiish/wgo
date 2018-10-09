package environ

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"wgo/wlog"
)

const (
	defaultType    = "file"
	defaultTag     = "WGO"
	defaultFormat  = "%T%E[%L] %M"
	defaultLevel   = "INFO|WARNING|ERROR|FATAL"
	defaultMaxsize = 1 << 28 // 256MB
	defaultDaily   = false
	defaultMaxDays = 30
	defaultMkdir   = true
)

type (
	logger struct {
		logs []wlog.LogConfig
		wlog.Logger
	}
)

/* {{{ func newLogger() *logger
 *
 */
func newLogger() *logger {
	return &logger{
		Logger: make(wlog.Logger),
	}
}

/* }}} */

/* {{{ Logger() *logger
 * get logger
 */
func Logger() *logger { return environ.Logger() }
func (env *Environ) Logger() *logger {
	if env.logger == nil { // init logger
		env.logger = newLogger()
	}
	return env.logger
}

/* {{{ func (env *Environ) Init(cfg *Config)
 * 通过日志配置初始化日志引擎
 */
func (l *logger) Init(cfg *Config) {
	if l.logs = BuildLogs(cfg); len(l.logs) > 0 {
		for _, log := range l.logs {
			if log.Format == "" {
				log.Format = defaultFormat // default format
			}
			if environ != nil {
				log.Location = environ.Location
			}
			l.Start(wlog.LogConfig(log))
		}
	} else {
		panic("[PANIC] start logger failed!")
	}
	// forbid debug
	if !environ.DebugMode {
		l.Remove(wlog.DEBUG)
	} else {
		l.Add(wlog.DEBUG)
	}
	// for app level
	if level == LVL_PRODUCTION {
		// 如果是生产环境, 则只放出error以上的日志
		l.Limit(wlog.ERROR)
	}
}

/* {{{ func LogConig(lc Log)
 *
 */
func LogConig(lc wlog.LogConfig) wlog.LogConfig {
	return wlog.LogConfig(lc)
}

/* }}} */

/* {{{ func BuildLogs(cfg *Config)
 *
 */
func BuildLogs(cfg *Config) []wlog.LogConfig {
	logs := make([]wlog.LogConfig, 0)
	if cfg.Get(CFG_KEY_LOGS) != nil {
		if err := cfg.UnmarshalKey(CFG_KEY_LOGS, &logs); err != nil {
			panic(err)
		}
	} else {
		logs = DefaultLogs()
	}
	return logs
}

/* }}} */

/* {{{ func DefaultLogs() Logs
 *
 */
func DefaultLogs() []wlog.LogConfig {
	log := wlog.LogConfig{
		Type:    defaultType,
		Tag:     defaultTag,
		Format:  defaultFormat,
		Level:   defaultLevel,
		Path:    filepath.Join(defaultWorkDir, "logs", "debug.log"),
		Maxsize: defaultMaxsize,
		Daily:   defaultDaily,
		MaxDays: defaultMaxDays,
		Mkdir:   defaultMkdir,
	}
	return []wlog.LogConfig{log}
}

/* }}} */

/* {{{ func (env *Environ) DenyConsole()
 * 禁止console, for daemonize
 */
func (env *Environ) DenyConsole() {
	env.Logger().DeleteFilter("console")
}

/* }}} */

/* {{{ func (env *Environ) AddConsole()
 *
 */
func (env *Environ) AddConsole() {
	fc := false // find console
	for _, log := range env.Logger().logs {
		if strings.ToLower(log.Type) == "console" {
			fc = true
			break
		}
	}
	if !fc { // not find console logging, add
		env.Logger().Start(wlog.LogConfig{
			Type:   "console",
			Tag:    "WGO",
			Format: defaultFormat,
			Level:  defaultLevel,
		})
	}
}

/* }}} */

// native log
func nlog(arg0 interface{}, args ...interface{}) {
	switch first := arg0.(type) {
	case string:
		// Use the first string as a format string
		log.Printf(first, args...)
	default:
		log.Printf(fmt.Sprint(first)+strings.Repeat(" %v", len(args)), args...)
	}
}

/* {{{ func Debug()
 *
 */
func (env *Environ) Debug(arg0 interface{}, args ...interface{}) {
	if env != nil && env.logger != nil {
		env.logger.Debug(arg0, args...)
	} else {
		nlog(arg0, args...)
	}
}

/* }}} */

/* {{{ func Info()
 *
 */
func (env *Environ) Info(arg0 interface{}, args ...interface{}) {
	if env != nil && env.logger != nil {
		env.logger.Info(arg0, args...)
	} else {
		nlog(arg0, args...)
	}
}

/* }}} */

/* {{{ func Warn()
 *
 */
func (env *Environ) Warn(arg0 interface{}, args ...interface{}) {
	if env != nil && env.logger != nil {
		env.logger.Warn(arg0, args...)
	} else {
		nlog(arg0, args...)
	}
}

/* }}} */

/* {{{ func Error()
 *
 */
func (env *Environ) Error(arg0 interface{}, args ...interface{}) {
	if env != nil && env.logger != nil {
		env.logger.Error(arg0, args...)
	} else {
		nlog(arg0, args...)
	}
}

/* }}} */

/* {{{ func Log()
 *
 */
func (env *Environ) Log(arg0 interface{}, args ...interface{}) {
	if env != nil && env.logger != nil {
		env.logger.Error(arg0, args...)
	} else {
		nlog(arg0, args...)
	}
}

/* }}} */

/* {{{ func Fatal()
 *
 */
func (env *Environ) Fatal(arg0 interface{}, args ...interface{}) {
	if env != nil && env.logger != nil {
		env.logger.Fatal(arg0, args...)
	} else {
		nlog(arg0, args...)
	}
	time.Sleep(10 * time.Millisecond)
	os.Exit(1)
}

/* }}} */
