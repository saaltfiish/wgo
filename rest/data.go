// Package rest provides ...
package rest

import (
	"database/sql"
	"math/bits"
	"strconv"
)

// JSON, 库里为字符串, struct 里为变量
type Array []interface{}

// func (a *Array) ToDb() (interface{}, error) {
// 	ab, err := json.Marshal(a)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return string(ab), err
// }
//
// func (a *Array) FromDb(target interface{}) (interface{}, func(interface{}, interface{}) error) {
// 	binder := func(holder, target interface{}) error {
// 		var js string
// 		if holder.(*sql.NullString).Valid {
// 			js = holder.(*sql.NullString).String
// 		}
// 		na := new(Array)
// 		if err := json.Unmarshal([]byte(js), na); err == nil {
// 			*(target.(**Array)) = na
// 		}
// 		return nil
// 	}
// 	return new(sql.NullString), binder
// }

// checklist, 按位记录状态
type Checklist map[string]bool
type ChecklistDic map[int]string

func (cl Checklist) Pack() int {
	sn := 0
	for key, t := range cl {
		offset, err := strconv.Atoi(key)
		if err == nil {
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
			cl[strconv.Itoa(offset)] = true
		} else {
			cl[strconv.Itoa(offset)] = false
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
func (cl Checklist) Translate(seq map[int]string) Checklist {
	tcl := make(Checklist)
	for offset, name := range seq {
		if t, ok := cl[strconv.Itoa(offset)]; ok {
			tcl[name] = t
		} else if t, ok := cl[name]; ok {
			tcl[strconv.Itoa(offset)] = t
		} else {
			tcl[strconv.Itoa(offset)] = false
			tcl[name] = false
		}
	}
	return tcl
}
