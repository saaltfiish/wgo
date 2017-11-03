// Package rest provides ...
package rest

import (
	"database/sql"
	"encoding/json"

	"gorp"
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

func (a *Array) FromDb(target interface{}) (gorp.CustomScanner, bool) {
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
	return gorp.CustomScanner{new(sql.NullString), target, binder}, true
}
