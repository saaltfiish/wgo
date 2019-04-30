//
// type.go
// Copyright (C) 2019 Odin <Odin@Odin-Pro.local>
//
// Distributed under terms of the MIT license.
//

package utils

import (
	"encoding/json"
	"strconv"
)

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
		default:
			vb, _ := json.Marshal(vi)
			return string(vb)
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

// second arg is default
func MustBool(opts ...interface{}) bool {
	params := NewParams(opts)
	vi := params.ItfByIndex(0)
	def := params.ItfByIndex(1)
	if bv, ok := vi.(bool); ok {
		return bv
	}
	if db, ok := def.(bool); ok {
		return db
	}
	return false
}

func MustArray(vi interface{}) []interface{} {
	if arr, ok := vi.([]interface{}); ok {
		return arr
	}
	rt := make([]interface{}, 0)
	if sa, ok := vi.([]string); ok {
		for _, s := range sa {
			rt = append(rt, s)
		}
		return rt
	}
	if ia, ok := vi.([]int); ok {
		for _, i := range ia {
			rt = append(rt, i)
		}
		return rt
	}
	if i64a, ok := vi.([]int64); ok {
		for _, i64 := range i64a {
			rt = append(rt, i64)
		}
		return rt
	}
	return nil
}

func MustStringMapString(vi interface{}) map[string]string {
	if vi != nil {
		if rt, ok := vi.(map[string]string); ok {
			return rt
		}
	}
	return nil
}

func MustStringMap(vi interface{}) map[string]interface{} {
	if vi != nil {
		if rt, ok := vi.(map[string]interface{}); ok {
			return rt
		}
	}
	return nil
}

// int64 pointer
func Int64Pointer(i interface{}) *int64 {
	rt := new(int64)
	*rt = MustInt64(i)
	return rt
}

// int pointer
func IntPointer(i interface{}) *int {
	rt := new(int)
	*rt = int(MustInt64(i))
	return rt
}

// int64 pointer
func StringPointer(i interface{}) *string {
	rt := new(string)
	*rt = MustString(i)
	return rt
}
