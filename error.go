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
