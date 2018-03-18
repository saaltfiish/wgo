//
// utils.go
// Copyright (C) 2018 Odin <Odin@Odin-Pro.local>
//
// Distributed under terms of the MIT license.
//

package utils

import (
	"github.com/davecgh/go-spew/spew"
)

func Dump(i ...interface{}) string {
	return spew.Sdump(i...)
}
