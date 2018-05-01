//
// json.go
// Copyright (C) 2018 Odin <Odin@Odin-Pro.local>
//
// Distributed under terms of the MIT license.
//

package utils

import (
	"github.com/bitly/go-simplejson"
)

type (
	Json struct {
		json *simplejson.Json
	}
)

func NewJson(body []byte) (*Json, error) {
	if json, err := simplejson.NewJson(body); err != nil {
		return nil, err
	} else {
		return &Json{json: json}, nil
	}
}

// func (j *Json) Array() ([]interface{}, error)
func (j *Json) Array() ([]interface{}, error) {
	return j.json.Array()
}

// func (j *Json) Bool() (bool, error)
func (j *Json) Bool() (bool, error) {
	return j.json.Bool()
}

// func (j *Json) Bytes() ([]byte, error)
func (j *Json) Bytes() ([]byte, error) {
	return j.json.Bytes()
}

// func (j *Json) CheckGet(key string) (*Json, bool)
func (j *Json) CheckGet(key string) (*Json, bool) {
	json, ok := j.json.CheckGet(key)
	return &Json{json: json}, ok
}

// func (j *Json) Del(key string)
func (j *Json) Del(key string) {
	j.json.Del(key)
}

// func (j *Json) Encode() ([]byte, error)
func (j *Json) Encode() ([]byte, error) {
	return j.json.Encode()
}

// func (j *Json) EncodePretty() ([]byte, error)
func (j *Json) EncodePretty() ([]byte, error) {
	return j.json.EncodePretty()
}

// func (j *Json) Float64() (float64, error)
func (j *Json) Float64() (float64, error) {
	return j.json.Float64()
}

// func (j *Json) Get(key string) *Json
func (j *Json) Get(key string) *Json {
	return &Json{json: j.json.Get(key)}
}

// func (j *Json) GetIndex(index int) *Json
func (j *Json) GetIndex(index int) *Json {
	return &Json{json: j.json.GetIndex(index)}
}

// func (j *Json) GetPath(branch ...string) *Json
func (j *Json) GetPath(branch ...string) *Json {
	return &Json{json: j.json.GetPath(branch...)}
}

// func (j *Json) Int() (int, error)
func (j *Json) Int() (int, error) {
	return j.json.Int()
}

// func (j *Json) Int64() (int64, error)
func (j *Json) Int64() (int64, error) {
	return j.json.Int64()
}

// func (j *Json) Interface() interface{}
func (j *Json) Interface() interface{} {
	return j.json.Interface()
}

// func (j *Json) Map() (map[string]interface{}, error)
func (j *Json) Map() (map[string]interface{}, error) {
	return j.json.Map()
}

// func (j *Json) MarshalJSON() ([]byte, error)
func (j *Json) MarshalJSON() ([]byte, error) {
	return j.json.MarshalJSON()
}

// func (j *Json) MustArray(args ...[]interface{}) []interface{}
func (j *Json) MustArray(args ...[]interface{}) []interface{} {
	return j.json.MustArray(args...)
}

// func (j *Json) MustBool(args ...bool) bool
func (j *Json) MustBool(args ...bool) bool {
	return j.json.MustBool(args...)
}

// func (j *Json) MustFloat64(args ...float64) float64
func (j *Json) MustFloat64(args ...float64) float64 {
	return j.json.MustFloat64(args...)
}

// func (j *Json) MustInt(args ...int) int
func (j *Json) MustInt(args ...int) int {
	return j.json.MustInt(args...)
}

// func (j *Json) MustInt64(args ...int64) int64
func (j *Json) MustInt64(args ...int64) int64 {
	return j.json.MustInt64(args...)
}

// func (j *Json) MustMap(args ...map[string]interface{}) map[string]interface{}
func (j *Json) MustMap(args ...map[string]interface{}) map[string]interface{} {
	return j.json.MustMap(args...)
}

// func (j *Json) MustString(args ...string) string
func (j *Json) MustString(args ...string) string {
	return j.json.MustString(args...)
}

// func (j *Json) MustStringArray(args ...[]string) []string
func (j *Json) MustStringArray(args ...[]string) []string {
	return j.json.MustStringArray(args...)
}

// func (j *Json) MustUint64(args ...uint64) uint64
func (j *Json) MustUint64(args ...uint64) uint64 {
	return j.json.MustUint64(args...)
}

// func (j *Json) Set(key string, val interface{})
func (j *Json) Set(key string, val interface{}) {
	j.json.Set(key, val)
}

// func (j *Json) SetPath(branch []string, val interface{})
func (j *Json) SetPath(branch []string, val interface{}) {
	j.json.SetPath(branch, val)
}

// func (j *Json) String() (string, error)
func (j *Json) String() (string, error) {
	return j.json.String()
}

// func (j *Json) StringArray() ([]string, error)
func (j *Json) StringArray() ([]string, error) {
	return j.json.StringArray()
}

// func (*Json) Uint64
func (j *Json) Uint64() (uint64, error) {
	return j.json.Uint64()
}

// func (j *Json) UnmarshalJSON(p []byte) error
func (j *Json) UnmarshalJSON(p []byte) error {
	return j.json.UnmarshalJSON(p)
}
