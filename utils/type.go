//
// type.go
// Copyright (C) 2019 Odin <Odin@Odin-Pro.local>
//
// Distributed under terms of the MIT license.
//

package utils

import (
	"encoding/json"
	"reflect"
	"strconv"
)

/* {{{ func ToType(i interface{}) reflect.Type
 * 如果是指针, 则调用Elem()至Type为止, 如果Type不是struct, 报错
 */
func ToType(i interface{}) reflect.Type {
	var t reflect.Type
	if tt, ok := i.(reflect.Type); ok {
		t = tt
	} else {
		t = reflect.TypeOf(i)
	}

	// If a Pointer to a type, follow
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	return t
}

/* }}} */

func MustString(vi interface{}) string {
	if vi != nil {
		switch v := vi.(type) {
		case string:
			return v
		case *string:
			if v != nil {
				return *v
			}
		case *int:
			if v != nil {
				return strconv.Itoa(*v)
			}
		case int:
			return strconv.Itoa(v)
		case *int64:
			if v != nil {
				return strconv.FormatInt(*v, 10)
			}
		case int64:
			return strconv.FormatInt(v, 10)
		case float64:
			return strconv.FormatFloat(v, 'f', 2, 64)
		case *float64:
			if v != nil {
				return strconv.FormatFloat(*v, 'f', 2, 64)
			}
		case json.Number:
			return v.String()
		default:
			vb, _ := json.Marshal(vi)
			return string(vb)
		}
	}
	return ""
}

func MustInt64(vi interface{}) int64 {
	if vi != nil { // 这个检测不到(*Ptr)(nil), 指针都在switch中检查
		switch v := vi.(type) {
		case string:
			v64, _ := strconv.ParseInt(v, 10, 64)
			return v64
		case *string:
			if v != nil {
				v64, _ := strconv.ParseInt(*v, 10, 64)
				return v64
			}
		case int:
			return int64(v)
		case *int:
			if v != nil {
				return int64(*v)
			}
		case int64:
			return v
		case *int64:
			if v != nil {
				return *v
			}
		case float64:
			return int64(v)
		case *float64:
			if v != nil {
				return int64(*v)
			}
		case json.Number:
			v64, _ := v.Int64()
			return v64
		}
	}
	return 0
}

func MustInt(vi interface{}) int {
	if vi != nil {
		switch v := vi.(type) {
		case string:
			v64, _ := strconv.Atoi(v)
			return v64
		case *string:
			if v != nil {
				v64, _ := strconv.Atoi(*v)
				return v64
			}
		case *int:
			if v != nil {
				return *v
			}
		case int:
			return v
		case *int64:
			if v != nil {
				return int(*v)
			}
		case int64:
			return int(v)
		case float64:
			return int(v)
		case *float64:
			if v != nil {
				return int(*v)
			}
		case json.Number:
			v64, _ := v.Int64()
			return int(v64)
		}
	}
	return 0
}

func MustFloat64(vi interface{}) float64 {
	if vi != nil {
		switch v := vi.(type) {
		case string:
			v64, _ := strconv.ParseFloat(v, 64)
			return v64
		case *string:
			if v != nil {
				v64, _ := strconv.ParseFloat(*v, 64)
				return v64
			}
		case *int:
			if v != nil {
				return float64(*v)
			}
		case int:
			return float64(v)
		case *int64:
			if v != nil {
				return float64(*v)
			}
		case int64:
			return float64(v)
		case float64:
			return v
		case *float64:
			if v != nil {
				return *v
			}
		case json.Number:
			v64, _ := v.Float64()
			return v64
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

// return a pointer with value
func Pointer(i interface{}) interface{} {
	if t := reflect.TypeOf(i); t.Kind() == reflect.Ptr {
		return i
	} else {
		np := reflect.New(ToType(t))
		reflect.Indirect(np).Set(reflect.ValueOf(i))
		return np.Interface()
	}
}
