package environ

import (
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
	defaultLevel   = "DEBUG|INFO|WARNING|ERROR|FATAL"
	defaultMaxsize = 1 << 28 // 256MB
	defaultDaily   = false
	defaultMaxDays = 30
	defaultMkdir   = true
)

type (
	Log wlog.LogConfig

	Logs []Log

	logger struct {
		logs Logs
		wlog.Logger
	}
)

var (
	logs Logs
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
				log.Format = "%T%E[%L] %M" // default format
			}
			l.Start(wlog.LogConfig(log))
		}
	} else {
		panic("[PANIC] start logger failed!")
	}
}

/* {{{ func LogConig(lc Log)
 *
 */
func LogConig(lc Log) wlog.LogConfig {
	return wlog.LogConfig(lc)
}

/* }}} */

/* {{{ func BuildLogs(cfg *Config)
 *
 */
func BuildLogs(cfg *Config) Logs {
	if cfg.Get(CFG_KEY_LOGS) != nil {
		logs = Logs{}
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
func DefaultLogs() Logs {
	log := Log{
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
	return Logs{log}
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
			Format: "%T%E[%C] %M",
			Level:  "INFO|WARNING|ERROR|FATAL",
		})
	}
}

/* }}} */

/* {{{ func Debug()
 *
 */
func (env *Environ) Debug(arg0 interface{}, args ...interface{}) {
	env.Logger().Debug(arg0, args...)
}

/* }}} */

/* {{{ func Info()
 *
 */
func (env *Environ) Info(arg0 interface{}, args ...interface{}) {
	env.Logger().Info(arg0, args...)
}

/* }}} */

/* {{{ func Warn()
 *
 */
func (env *Environ) Warn(arg0 interface{}, args ...interface{}) {
	env.Logger().Warn(arg0, args...)
}

/* }}} */

/* {{{ func Error()
 *
 */
func (env *Environ) Error(arg0 interface{}, args ...interface{}) {
	env.Logger().Error(arg0, args...)
}

/* }}} */

/* {{{ func Log()
 *
 */
func (env *Environ) Log(arg0 interface{}, args ...interface{}) {
	env.Logger().Error(arg0, args...)
}

/* }}} */

/* {{{ func Fatal()
 *
 */
func (env *Environ) Fatal(arg0 interface{}, args ...interface{}) {
	env.Logger().Fatal(arg0, args...)
	time.Sleep(10 * time.Millisecond)
	os.Exit(1)
}

/* }}} */
