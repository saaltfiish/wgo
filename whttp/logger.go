// Package whttp provides ...
package whttp

import "wgo/wlog"

type (
	// logger
	Logger interface {
		Debug(interface{}, ...interface{})
		Info(interface{}, ...interface{})
		Warn(interface{}, ...interface{})
		Error(interface{}, ...interface{})
	}
)

var logger Logger

// whttp logger
func SetLogger(l Logger) {
	logger = l
}

// Debug
func Debug(arg0 interface{}, args ...interface{}) {
	if logger != nil {
		logger.Debug(arg0, args...)
	} else {
		wlog.Output(arg0, args...)
	}
}

// Info
func Info(arg0 interface{}, args ...interface{}) {
	if logger != nil {
		logger.Info(arg0, args...)
	} else {
		wlog.Output(arg0, args...)
	}
}

// Warn
func Warn(arg0 interface{}, args ...interface{}) {
	if logger != nil {
		logger.Warn(arg0, args...)
	} else {
		wlog.Output(arg0, args...)
	}
}

// Error
func Error(arg0 interface{}, args ...interface{}) {
	if logger != nil {
		logger.Error(arg0, args...)
	} else {
		wlog.Output(arg0, args...)
	}
}
