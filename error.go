//
// error.go
// Copyright (C) 2019 Odin <Odin@Odin-Pro.local>
//
// Distributed under terms of the MIT license.
//

package wgo

import "wgo/server"

func Errorf(code int, format string, a ...interface{}) *server.ServerError {
	return server.NewErrorf(code, format, a...)
}

// exract error code
func ErrorCode(err error) int64 {
	if se, ok := err.(*server.ServerError); ok {
		return int64(se.Status())
	}
	return -1
}
