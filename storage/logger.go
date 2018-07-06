//
// logger.go
// Copyright (C) 2018 Odin <Odin@Odin-Pro.local>
//
// Distributed under terms of the MIT license.
//

package storage

import (
	"fmt"
	"log"
	"strings"
)

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

func SetLogger(l Logger) {
	logger = l
}

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

// Debug
func Debug(arg0 interface{}, args ...interface{}) {
	if logger != nil {
		logger.Debug(arg0, args...)
	} else {
		nlog(arg0, args...)
	}
}

// Info
func Info(arg0 interface{}, args ...interface{}) {
	if logger != nil {
		logger.Info(arg0, args...)
	} else {
		nlog(arg0, args...)
	}
}

// Warn
func Warn(arg0 interface{}, args ...interface{}) {
	if logger != nil {
		logger.Warn(arg0, args...)
	} else {
		nlog(arg0, args...)
	}
}

// Error
func Error(arg0 interface{}, args ...interface{}) {
	if logger != nil {
		logger.Error(arg0, args...)
	} else {
		nlog(arg0, args...)
	}
}
