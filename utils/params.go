//
// params.go
// Copyright (C) 2019 Odin <Odin@Odin-Pro.local>
//
// Distributed under terms of the MIT license.
//
// 参数解析, 专门处理`...interface{}`参数

package utils

import (
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
		case *string:
			p.primaryStringKey = *pv
		case int64:
			p.primaryInt64Key = pv
		case int:
			p.primaryInt64Key = int64(pv)
		case *int64:
			p.primaryInt64Key = *pv
		case *int:
			p.primaryInt64Key = int64(*pv)
		case map[string]interface{}:
			for k, v := range pv {
				p.params[strings.ToLower(k)] = v
			}
		}
	}
	return p
}

func (p *Params) PrimaryInt64Key() int64 {
	return p.primaryInt64Key
}

func (p *Params) PrimaryStringKey() string {
	return p.primaryStringKey
}

func (p *Params) String(key string) string {
	if vi, ok := p.params[strings.ToLower(key)]; ok {
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

func (p *Params) Int64(key string) int64 {
	if vi, ok := p.params[strings.ToLower(key)]; ok {
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

func (p *Params) Destructuring(mp map[string]interface{}) map[string]interface{} {
	if len(mp) > 0 {
		for k, _ := range mp {
			if v, ok := p.params[strings.ToLower(k)]; ok {
				mp[k] = v
			}
		}
	}
	return mp
}
