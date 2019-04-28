//
// type.go
// Copyright (C) 2019 Odin <Odin@Odin-Pro.local>
//
// Distributed under terms of the MIT license.
//

package utils

import "strconv"

func MustString(vi interface{}) string {
	if vi != nil {
		switch v := vi.(type) {
		case string:
			return v
		case *string:
			return *v
		case *int:
			return strconv.Itoa(*v)
		case int:
			return strconv.Itoa(v)
		case *int64:
			return strconv.FormatInt(*v, 10)
		case int64:
			return strconv.FormatInt(v, 10)
		}
	}
	return ""
}

func MustInt64(vi interface{}) int64 {
	if vi != nil {
		switch v := vi.(type) {
		case string:
			v64, _ := strconv.ParseInt(v, 10, 64)
			return v64
		case *string:
			v64, _ := strconv.ParseInt(*v, 10, 64)
			return v64
		case *int:
			return int64(*v)
		case int:
			return int64(v)
		case *int64:
			return *v
		case int64:
			return v
		}
	}
	return 0
}

// int64 pointer
func Int64Pointer(i int64) *int64 {
	rt := new(int64)
	*rt = i
	return rt
}
