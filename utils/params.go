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
				p.params[strings.ToLower(k)] = v
			}
		}
	}
	return p
}

func PrimaryInt64Key(p ...interface{}) int64 {
	return NewParams(p).PrimaryInt64Key()
}

func (p *Params) PrimaryInt64Key() int64 {
	return p.primaryInt64Key
}

func PrimaryStringKey(p ...interface{}) string {
	return NewParams(p).PrimaryStringKey()
}

func (p *Params) PrimaryStringKey() string {
	return p.primaryStringKey
}

func (p *Params) Bind(ptr interface{}) error {
	return Bind(ptr, p.params)
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
