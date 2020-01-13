//
// params.go
// Copyright (C) 2019 Odin <Odin@Odin-Pro.local>
//
// Distributed under terms of the MIT license.
//
// 参数解析, 专门处理`...interface{}`参数

package utils

import (
	"errors"
	"reflect"
	"strings"
)

type Params struct {
	origin []interface{}
	params map[string]interface{}
}

func NewParams(p []interface{}) *Params {
	params := &Params{origin: p, params: make(map[string]interface{})}
	return params.Parse()
}

func (p *Params) Remains(off int) []interface{} {
	// 代表第off个之后的所有参数
	if p.Len() <= off+1 {
		return nil
	}
	return p.origin[off+1:]
}

func (p *Params) Parse() *Params {
	if p.origin != nil && len(p.origin) >= 1 && len(p.origin)%2 == 0 {
		// 分析第一个参数
		primary := p.origin[0]
		if pv, ok := primary.(map[string]interface{}); ok {
			// 第一个参数以`map[string]interface{}`传入
			for k, v := range pv {
				switch rv := reflect.ValueOf(v); rv.Kind() {
				case reflect.Ptr, reflect.Slice, reflect.Chan, reflect.Interface, reflect.Func, reflect.Map:
					// 如果不是以上类型, IsNil会panic
					if !rv.IsNil() {
						p.params[strings.ToLower(k)] = v
					}
				default:
					p.params[strings.ToLower(k)] = v
				}
			}
		}
		// even number params, odd is name, even is value
		for i := 0; i < len(p.origin); i += 2 {
			if name, ok := p.origin[i].(string); ok && name != "" {
				switch rv := reflect.ValueOf(p.origin[i+1]); rv.Kind() {
				case reflect.Ptr, reflect.Slice, reflect.Chan, reflect.Interface, reflect.Func, reflect.Map:
					if !rv.IsNil() {
						p.params[strings.ToLower(name)] = p.origin[i+1]
					}
				default:
					p.params[strings.ToLower(name)] = p.origin[i+1]
				}
			}
		}
	}
	return p
}

// primary, 只有仅传入一个参数时有效, 如果想找第一个参数可以用`*ByIndex()`系列
func PrimaryInt64(ps []interface{}) int64 {
	return NewParams(ps).PrimaryInt64()
}
func (p *Params) PrimaryInt64() int64 {
	if pl := p.Len(); pl == 1 {
		return p.Int64ByIndex(0)
	}
	return 0
}

func PrimaryInt(ps []interface{}) int {
	return NewParams(ps).PrimaryInt()
}
func (p *Params) PrimaryInt() int {
	if pl := p.Len(); pl == 1 {
		return p.IntByIndex(0)
	}
	return 0
}

func PrimaryString(ps []interface{}) string {
	return NewParams(ps).PrimaryString()
}
func (p *Params) PrimaryString() string {
	if pl := p.Len(); pl == 1 {
		return p.StringByIndex(0)
	}
	return ""
}

func PrimaryInterface(ps []interface{}) interface{} {
	return NewParams(ps).PrimaryInterface()
}
func (p *Params) PrimaryInterface() interface{} {
	if pl := p.Len(); pl == 1 {
		return p.ItfByIndex(0)
	}
	return nil
}

// last param, opts len must bigger than 1
func LastInt64(ps []interface{}) int64 {
	return NewParams(ps).LastInt64()
}
func (p *Params) LastInt64() int64 {
	if pl := p.Len(); pl > 1 {
		return p.Int64ByIndex(pl - 1)
	}
	return 0
}

func LastInt(ps []interface{}) int {
	return NewParams(ps).LastInt()
}
func (p *Params) LastInt() int {
	if pl := p.Len(); pl > 1 {
		return p.IntByIndex(pl - 1)
	}
	return 0
}

func LastString(ps []interface{}) string {
	return NewParams(ps).LastString()
}
func (p *Params) LastString() string {
	if pl := p.Len(); pl > 1 {
		return p.StringByIndex(pl - 1)
	}
	return ""
}

func LastInterface(ps []interface{}) interface{} {
	return NewParams(ps).LastInterface()
}
func (p *Params) LastInterface() interface{} {
	if pl := p.Len(); pl > 1 {
		return p.ItfByIndex(pl - 1)
	}
	return nil
}

func LastBool(ps []interface{}, def bool) bool {
	return NewParams(ps).LastBool(def)
}
func (p *Params) LastBool(def bool) bool {
	if pl := p.Len(); pl > 1 {
		return p.BoolByIndex(pl-1, def)
	}
	return def
}

// 参数长度
func (p *Params) Len() int {
	if p.origin != nil {
		return len(p.origin)
	}
	return 0
}

// shift, 移除第一个参数
func (p *Params) Shift() *Params {
	if p.Len() > 0 {
		return NewParams(p.origin[1:])
	}
	return p
}

func (p *Params) Bind(ptr interface{}) error {
	return Bind(ptr, p.params)
}

// 通过key获取, 适用于传入一个map[string]interface{}的情况
// 返回直接返回interface
func (p *Params) Interface(key string) interface{} {
	return p.Itf(key)
}
func (p *Params) Itf(key string) interface{} {
	if vi, ok := p.params[strings.ToLower(key)]; ok {
		return vi
	}
	return nil
}

// 通过key获取, 适用于传入一个map[string]interface{}的情况
// 努力尝试返回string
func (p *Params) String(key string) string {
	return MustString(p.Itf(key))
}

// 通过key获取, 适用于传入一个map[string]interface{}的情况
// 努力尝试返回int64
func (p *Params) Int64(key string) int64 {
	return MustInt64(p.Itf(key))
}

// 通过key获取, 适用于传入一个map[string]interface{}的情况
// 努力尝试返回bool
func (p *Params) Bool(key string, opts ...bool) bool {
	def := false
	switch len(opts) {
	case 1:
		def = opts[0]
	}
	iv := p.Itf(key)
	if rt := MustBool(iv, def); !rt && !def {
		// 传入的不是bool的时候, 尝试"yes", "true"
		sb := strings.ToLower(MustString(iv))
		return sb == "yes" || sb == "true"
	} else {
		return rt
	}
}

// 通过key获取, 适用于传入一个map[string]interface{}的情况
// 努力尝试返回[]interface{}
func (p *Params) Array(key string) []interface{} {
	if ai := p.Itf(key); ai != nil {
		return MustArray(ai)
	}
	return nil
}

// 通过key获取
// 尝试返回map[string]interface{}
func (p *Params) StringMap(key string) map[string]interface{} {
	if ai := p.Itf(key); ai != nil {
		return MustStringMap(ai)
	}
	return nil
}

// 通过key获取
// 尝试返回map[string]string
func (p *Params) StringMapString(key string) map[string]string {
	if ai := p.Itf(key); ai != nil {
		return MustStringMapString(ai)
	}
	return nil
}

// 通过下标获取interface{}
func (p *Params) InterfaceByIndex(offset int) interface{} {
	return p.ItfByIndex(offset)
}
func (p *Params) ItfByIndex(offset int) interface{} {
	if len(p.origin) > offset {
		return p.origin[offset]
	}
	return nil
}

// 通过下标获取string(适用于传入多个参数的情况), 0-based
func (p *Params) StringByIndex(offset int) string {
	return MustString(p.ItfByIndex(offset))
}

// 通过下标获取int64(适用于传入多个参数的情况), 0-based
func (p *Params) Int64ByIndex(offset int) int64 {
	return MustInt64(p.ItfByIndex(offset))
}

// 通过下标获取int64指针(适用于传入多个参数的情况), 0-based
func (p *Params) Int64PointerByIndex(offset int) *int64 {
	if i := p.ItfByIndex(offset); i != nil {
		return Int64Pointer(i)
	}
	return nil
}

// 通过下标获取int(适用于传入多个参数的情况), 0-based
func (p *Params) IntByIndex(offset int) int {
	return MustInt(p.ItfByIndex(offset))
}

// 通过下标获取int指针(适用于传入多个参数的情况), 0-based
func (p *Params) IntPointerByIndex(offset int) *int {
	if i := p.ItfByIndex(offset); i != nil {
		return IntPointer(i)
	}
	return nil
}

func (p *Params) BoolByIndex(offset int, opts ...bool) bool {
	def := false
	switch len(opts) {
	case 1:
		def = opts[0]
	}
	iv := p.ItfByIndex(offset)
	if rt := MustBool(iv, def); !rt && !def {
		// 传入的不是bool的时候, 尝试"yes", "true"
		sb := strings.ToLower(MustString(iv))
		return sb == "yes" || sb == "true"
	} else {
		return rt
	}
}

func (p *Params) ArrayByIndex(offset int) []interface{} {
	if ai := p.ItfByIndex(offset); ai != nil {
		return MustArray(ai)
	}
	return nil
}

func (p *Params) StringMapByIndex(offset int) map[string]interface{} {
	if ai := p.ItfByIndex(offset); ai != nil {
		return MustStringMap(ai)
	}
	return nil
}

func (p *Params) StringMapStringByIndex(offset int) map[string]string {
	if ai := p.ItfByIndex(offset); ai != nil {
		return MustStringMapString(ai)
	}
	return nil
}

func (p *Params) Destructuring(mp map[string]interface{}) error {
	matched := false
	if len(mp) > 0 {
		for k, _ := range mp {
			if v, ok := p.params[strings.ToLower(k)]; ok {
				matched = true
				mp[k] = v
			}
		}
		if matched {
			return nil
		}
	}
	return errors.New("[Destructuring]failed")
}
