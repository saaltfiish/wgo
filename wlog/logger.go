package wlog

import (
	"fmt"
	"os"
	"strings"
	"time"
)

const (
	LogBufferLength = 20480

	DEBUG = iota
	TRACE
	INFO
	WARNING
	ERROR
	CRITICAL
	FATAL
	ACCESS
)

// Logging level strings
var (
	// levelMapping
	levelMapping = map[string]uint{
		"DEBUG":    DEBUG,
		"TRACE":    TRACE,
		"INFO":     INFO,
		"WARNING":  WARNING,
		"ERROR":    ERROR,
		"CRITICAL": CRITICAL,
		"FATAL":    FATAL,
		"ACCESS":   ACCESS,
	}

	pid int
)

func init() {
	pid = os.Getpid()
}

type (
	LogConfig struct {
		Type     string `mapstructure:"type"`
		Tag      string `mapstructure:"tag"`
		Format   string `mapstructure:"format"`
		Level    string `mapstructure:"level"`
		Path     string `mapstructure:"path"`
		Addr     string `mapstructure:"addr"`
		Network  string `mapstructure:"network"`
		Prefix   string `mapstructure:"prefix"`
		Thread   int    `mapstructure:"thread_num"`
		Filename string `mapstructure:"filename"`
		Maxsize  int    `mapstructure:"max_size"`
		Daily    bool   `mapstructure:"daily"`
		MaxDays  int    `mapstructure:"max_days"`
		Mkdir    bool   `mapstructure:"mkdir"`
		Location *time.Location
	}

	Level struct {
		offset int
		lvl    uint
		desc   string
	}

	// LogRecord 日志结构
	LogRecord struct {
		Tag     string
		Level   Level     // The log level
		Created time.Time // The time at which the log message was created (nanoseconds)
		Message string    // The log message
	}

	// LogWriter writer 接口
	LogWriter interface {
		LogWrite(rec *LogRecord)
		Close()
	}

	// Filter filter
	Filter struct {
		Tag      string
		Type     string
		Level    Level
		Location *time.Location
		//LevelStr string
		LogWriter
	}

	Logger map[string]*Filter
)

func (log Logger) Close() {
	for name, filter := range log {
		filter.Close()
		delete(log, name)
	}
}

func (log Logger) Start(cfg LogConfig) {
	var writer LogWriter
	switch strings.ToLower(cfg.Type) {
	case "console":
		writer = NewConsoleLogWriter(cfg.Format)
	case "file":
		writer = NewFileLogWriter(
			cfg.Format,
			cfg.Path,
			cfg.Mkdir,
			cfg.Daily,
			cfg.Maxsize,
			cfg.MaxDays,
		)
	case "syslog":
		writer = NewSysLogWriter(
			cfg.Format,
			cfg.Network,
			cfg.Addr,
			cfg.Thread,
			cfg.Prefix,
		)
	default:
	}
	if writer != nil {
		//log.AddFilter(
		//	cfg.Tag,
		//	cfg.Type,
		//	BuildLevel(cfg.Level),
		//	writer,
		//)
		log[cfg.Tag] = &Filter{Tag: cfg.Tag, Type: strings.ToLower(cfg.Type), Level: BuildLevel(cfg.Level), Location: cfg.Location, LogWriter: writer}
	}

}

//func (log Logger) AddFilter(name string, typ string, lvl Level, writer LogWriter) Logger {
//	log[name] = &Filter{Tag: name, Type: strings.ToLower(typ), Level: lvl, LogWriter: writer}
//	return log
//}

func (log Logger) DeleteFilter(typ string) {
	for name, filter := range log {
		if strings.ToLower(filter.Type) == typ {
			filter.Close()
			delete(log, name)
		}
	}
}

func (log Logger) intLogf(lvl Level, arg0 interface{}, args ...interface{}) (rec *LogRecord) {

	for _, filter := range log {
		if r := filter.Level.lvl & lvl.lvl; r == 0 {
			continue
		}

		var msg string
		if len(args) > 0 {
			switch first := arg0.(type) {
			case string:
				// Use the first string as a format string
				msg = fmt.Sprintf(first, args...)
			default:
				msg = fmt.Sprintf(fmt.Sprint(first)+strings.Repeat(" %v", len(args)), args...)
			}
		} else if m, ok := arg0.(string); ok && m != "" {
			msg = m
		}

		// Make the log record
		created := time.Now()
		if filter.Location != nil {
			created = created.In(filter.Location)
		}
		rec = &LogRecord{
			Tag:     filter.Tag,
			Level:   lvl,
			Created: created,
			Message: msg,
		}

		filter.LogWrite(rec)
	}

	return
}

func (log Logger) Debug(arg0 interface{}, args ...interface{}) *LogRecord {
	return log.intLogf(Level{offset: DEBUG, lvl: 1 << DEBUG, desc: "DEBUG"}, arg0, args...)
}

func (log Logger) Trace(arg0 interface{}, args ...interface{}) *LogRecord {
	return log.intLogf(Level{offset: TRACE, lvl: 1 << TRACE, desc: "TRACE"}, arg0, args...)
}

func (log Logger) Info(arg0 interface{}, args ...interface{}) *LogRecord {
	return log.intLogf(Level{offset: INFO, lvl: 1 << INFO, desc: "INFO"}, arg0, args...)
}

func (log Logger) Warn(arg0 interface{}, args ...interface{}) *LogRecord {
	return log.intLogf(Level{offset: WARNING, lvl: 1 << WARNING, desc: "WARN"}, arg0, args...)
}

func (log Logger) Error(arg0 interface{}, args ...interface{}) *LogRecord {
	return log.intLogf(Level{offset: ERROR, lvl: 1 << ERROR, desc: "ERROR"}, arg0, args...)
}

func (log Logger) Critical(arg0 interface{}, args ...interface{}) *LogRecord {
	return log.intLogf(Level{offset: CRITICAL, lvl: 1 << CRITICAL, desc: "CRITICAL"}, arg0, args...)
}

func (log Logger) Fatal(arg0 interface{}, args ...interface{}) *LogRecord {
	return log.intLogf(Level{offset: FATAL, lvl: 1 << FATAL, desc: "FATAL"}, arg0, args...)
}

func (log Logger) Access(arg0 interface{}, args ...interface{}) *LogRecord {
	return log.intLogf(Level{offset: ACCESS, lvl: 1 << ACCESS, desc: "ACCESS"}, arg0, args...)
}

// BuildLevel 根据字符串转化为level
func BuildLevel(lvl string) Level {
	lss := strings.Split(lvl, "|")
	num := 0
	for _, l := range lss {
		if offset, ok := levelMapping[strings.ToUpper(l)]; ok {
			num = num | (1 << offset)
		}
	}
	return Level{
		lvl:  uint(num),
		desc: lvl,
	}
}

// add levels
func (log Logger) Add(ls ...int) {
	for _, filter := range log {
		for _, l := range ls {
			for _, lvl := range levelMapping {
				if lvl == uint(l) {
					filter.Level.lvl = filter.Level.lvl | (1 << lvl)
				}
			}
		}
	}
}

// remove levels
func (log Logger) Remove(ls ...int) {
	for _, filter := range log {
		for _, l := range ls {
			for _, lvl := range levelMapping {
				if lvl == uint(l) {
					filter.Level.lvl = filter.Level.lvl &^ (1 << lvl)
				}
			}
		}
	}
}

// level limit, 低于某个level的都不要
func (log Logger) Limit(l int) {
	for _, filter := range log {
		for _, lvl := range levelMapping {
			if lvl < uint(l) {
				filter.Level.lvl = filter.Level.lvl &^ (1 << lvl)
			}
		}
	}
}
