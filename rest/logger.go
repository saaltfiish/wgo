// Package rest provides ...
package rest

import (
	"wgo"
)

type (
	// logger
	Logger interface {
		Debug(interface{}, ...interface{})
		Info(interface{}, ...interface{})
		Warn(interface{}, ...interface{})
		Error(interface{}, ...interface{})
		Printf(string, ...interface{})
	}
)

var logger Logger

func SetLogger(l Logger) {
	logger = l
}

// Debug
func Debug(arg0 interface{}, args ...interface{}) {
	if logger != nil {
		logger.Debug(arg0, args...)
	} else {
		wgo.Debug(arg0, args...)
	}
}

// Info
func Info(arg0 interface{}, args ...interface{}) {
	if logger != nil {
		logger.Info(arg0, args...)
	} else {
		wgo.Info(arg0, args...)
	}
}

// Warn
func Warn(arg0 interface{}, args ...interface{}) {
	if logger != nil {
		logger.Warn(arg0, args...)
	} else {
		wgo.Warn(arg0, args...)
	}
}

// Error
func Error(arg0 interface{}, args ...interface{}) {
	if logger != nil {
		logger.Error(arg0, args...)
	} else {
		wgo.Error(arg0, args...)
	}
}

// logging
// Debug
func (rest *REST) Debug(arg0 interface{}, args ...interface{}) {
	if rest.Context() != nil {
		rest.Context().Debug(arg0, args...)
	} else {
		wgo.Debug(arg0, args...)
	}
}

// Info
func (rest *REST) Info(arg0 interface{}, args ...interface{}) {
	if rest.Context() != nil {
		rest.Context().Info(arg0, args...)
	} else {
		wgo.Info(arg0, args...)
	}
}

// Warn
func (rest *REST) Warn(arg0 interface{}, args ...interface{}) {
	if rest.Context() != nil {
		rest.Context().Warn(arg0, args...)
	} else {
		wgo.Warn(arg0, args...)
	}
}

// Error
func (rest *REST) Error(arg0 interface{}, args ...interface{}) {
	if rest.Context() != nil {
		rest.Context().Error(arg0, args...)
	} else {
		wgo.Error(arg0, args...)
	}
}
