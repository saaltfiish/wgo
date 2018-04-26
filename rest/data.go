// Package rest provides ...
package rest

import (
	"database/sql"
	"encoding/json"
	"math/bits"
	"strconv"
)

// JSON, 库里为字符串, struct 里为变量
type Array []interface{}

func (a *Array) ToDb() (interface{}, error) {
	ab, err := json.Marshal(a)
	if err != nil {
		return nil, err
	}
	return string(ab), err
}

func (a *Array) FromDb(target interface{}) (interface{}, func(interface{}, interface{}) error) {
	binder := func(holder, target interface{}) error {
		var js string
		if holder.(*sql.NullString).Valid {
			js = holder.(*sql.NullString).String
		}
		na := new(Array)
		if err := json.Unmarshal([]byte(js), na); err == nil {
			*(target.(**Array)) = na
		}
		return nil
	}
	return new(sql.NullString), binder
}

// checklist, 按位记录状态
type Checklist map[interface{}]bool

func (cl Checklist) Pack() int {
	sn := 0
	for key, t := range cl {
		offset, ok := key.(int) // key必须转为int才能pack
		if ok {
			if t {
				sn = sn | (1 << uint(offset))
			} else {
				sn = sn &^ (1 << uint(offset))
			}
		}
	}
	return sn
}

func (cl Checklist) Unpack(sn int) Checklist {
	for offset := 0; offset < bits.Len(uint(sn)); offset++ {
		if sn&(1<<uint(offset)) > 0 {
			cl[offset] = true
		} else {
			cl[offset] = false
		}
	}
	return cl
}

func (cl Checklist) ToDb() (interface{}, error) {
	return cl.Pack(), nil
}

func (cl Checklist) FromDb(target interface{}) (interface{}, func(interface{}, interface{}) error) {
	binder := func(holder, target interface{}) error {
		sn := 0
		if holder.(*sql.NullString).Valid {
			sns := holder.(*sql.NullString).String
			sn, _ = strconv.Atoi(sns)
		}
		ncl := make(Checklist)
		*(target.(*Checklist)) = ncl.Unpack(sn)
		return nil
	}
	return new(sql.NullString), binder
}

// translate
func (cl Checklist) Translate(seq map[int]string) map[string]bool {
	tcl := make(map[string]bool)
	for offset, name := range seq {
		if t, ok := cl[offset]; ok {
			tcl[name] = t
		} else {
			tcl[name] = false
		}
	}
	return tcl
}

// convert
// func ToChecklist(
