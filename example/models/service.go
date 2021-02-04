//
// service.go
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
type Service struct {
	Id       *string    `json:"id,omitempty" db:",na,pk" filter:"uuid,G"`               // service id`
	Name     *string    `json:"name,omitempty" db:",k" filter:",R,C"`            // service name
	Env      *string    `json:"env,omitemptye" db:",k" filter:",C"`            // service env parameter
	Modified *time.Time `json:"modified,omitempty" db:",ro"`                       // modified time
	Stage    *int       `json:"stage,omitempty" db:",logic" filter:",C"`           // status
	Config   *string    `json:"config,omitempty" db:",logic"`      // service config
	
	*rest.REST
}

func init() {
	rest.AddModel((*Service)(nil))
}
