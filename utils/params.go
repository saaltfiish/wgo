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
	"strconv"
	"strings"
)

type Params struct {
	origin           []interface{}
	params           map[string]interface{}
	primaryStringKey string
	primaryInt64Key  int64
}

func NewParams(p []interface{}) *Params {
	params := &Params{origin: p, params: make(map[string]interface{})}
	return params.Parse()
}

func (p *Params) Parse() *Params {
	if p.origin != nil && len(p.origin) == 1 {
		// 只传入了一个参数
		primary := p.origin[0]
		switch pv := primary.(type) {
		case string:
			p.primaryStringKey = pv
			p.primaryInt64Key, _ = strconv.ParseInt(pv, 10, 64)
		case *string:
			p.primaryStringKey = *pv
			p.primaryInt64Key, _ = strconv.ParseInt(*pv, 10, 64)
		case int64:
			p.primaryStringKey = strconv.FormatInt(pv, 10)
			p.primaryInt64Key = pv
		case int:
			p.primaryStringKey = strconv.FormatInt(int64(pv), 10)
			p.primaryInt64Key = int64(pv)
		case *int64:
			p.primaryStringKey = strconv.FormatInt(*pv, 10)
			p.primaryInt64Key = *pv
		case *int:
			p.primaryStringKey = strconv.FormatInt(int64(*pv), 10)
			p.primaryInt64Key = int64(*pv)
		case map[string]interface{}:
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
	} else if p.origin != nil && len(p.origin) > 0 && len(p.origin)%2 == 0 {
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

func PrimaryStringKey(ps []interface{}) string {
	return NewParams(ps).PrimaryStringKey()
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
