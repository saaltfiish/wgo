//
// runtime.go
// Copyright (C) 2020 Odin <Odin@Odin-Pro.local>
//
// Distributed under terms of the MIT license.
//

package utils

import (
	"path"
	"runtime"
	"strings"
)

type CallInfo struct {
	FullName    string
	FileName    string
	FuncName    string
	PackageName string
	MainPackage string
	Line        int
}

func RetrieveCallInfo(opts ...interface{}) *CallInfo {
	depth := 2
	if d := NewParams(opts).IntByIndex(0); d > 0 {
		depth = d
	}
	pc, file, line, _ := runtime.Caller(depth)
	_, fileName := path.Split(file)
	name := runtime.FuncForPC(pc).Name()
	parts := strings.Split(name, ".")
	pl := len(parts)
	packageName := ""

	n := 1
	funcName := parts[pl-n]
	if Numberic(funcName) {
		n = 2
		funcName = strings.Join(parts[pl-n:], ".")
	}

	if parts[pl-n-1][0] == '(' {
		funcName = parts[pl-n-1] + "." + funcName
		packageName = strings.Join(parts[0:pl-n-1], ".")
	} else {
		packageName = strings.Join(parts[0:pl-n], ".")
	}
	mpkg, _ := path.Split(packageName)
	mainPackage := strings.TrimSuffix(mpkg, "/")

	return &CallInfo{
		FullName:    name,
		PackageName: packageName,
		MainPackage: mainPackage,
		FileName:    fileName,
		FuncName:    funcName,
		Line:        line,
	}
}
