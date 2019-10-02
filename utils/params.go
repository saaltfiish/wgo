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
	origin           []interface{}
	params           map[string]interface{}
	primaryStringKey string
	primaryInt64Key  int64
	primaryIntKey    int
}

func NewParams(p []interface{}) *Params {
	params := &Params{origin: p, params: make(map[string]interface{})}
	return params.Parse()
}

func (p *Params) Parse() *Params {
	if p.origin != nil && len(p.origin) == 1 {
		// 只分析第一个参数
		primary := p.origin[0]
		if pv, ok := primary.(map[string]interface{}); ok {
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
		} else {
			p.primaryStringKey = MustString(primary)
			p.primaryInt64Key = MustInt64(primary)
			p.primaryIntKey = MustInt(primary)
		}
	} else if p.origin != nil && len(p.origin) > 1 && len(p.origin)%2 == 0 {
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

func PrimaryInt64Key(ps []interface{}) int64 {
	return NewParams(ps).PrimaryInt64Key()
}

func (p *Params) PrimaryInt64Key() int64 {
	return p.primaryInt64Key
}

func PrimaryIntKey(ps []interface{}) int {
	return NewParams(ps).PrimaryIntKey()
}

func (p *Params) PrimaryIntKey() int {
	return p.primaryIntKey
}

func PrimaryStringKey(ps []interface{}) string {
	return NewParams(ps).PrimaryStringKey()
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

func (p *Params) PrimaryStringKey() string {
	return p.primaryStringKey
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
