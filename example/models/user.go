//
// user.go
// Copyright (C) 2021 Odin <odinmanlee@gmail.com>
//
// Distributed under terms of the MIT license.
//

package models

import (
	"time"

	"wgo/rest"
)

// 以下struct需要跟数据库映射
type User struct {
	Id       *string    `json:"id,omitempty" db:",pk" filter:",C,G"`
	Name     *string    `json:"name,omitempty" db:",k" filter:",R,C,D"`           // Account Name
	Avatar   string     `json:"avatar,omitempty" filter:",C"`                     // avatar
	Stage    *int       `json:"stage,omitempty" db:",logic" filter:",C"`          // status
	Creator  *string    `json:"creator,omitempty" filter:"userid,C,D"`            // creator
	Created  *time.Time `json:"created,omitempty" db:",add_now" filter:",G,D,TR"` // created time
	Modified *time.Time `json:"modified,omitempty" db:",ro"`                      // modified time

	*rest.REST
}

func init() {
	rest.AddModel((*User)(nil))
}
