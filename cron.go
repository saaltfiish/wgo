//
// cron.go
// Copyright (C) 2018 Odin <Odin@Odin-Pro.local>
//
// Distributed under terms of the MIT license.
//

package wgo

import "wgo/cron"

// init
func initCron() {
	c := cron.NewWithLocation(Env().Location).Register()
	SetCron(c)
	c.Start()
}

// get cron
func Cron() *cron.Cron { return wgo.Cron() }
func (w *WGO) Cron() *cron.Cron {
	return w.cron
}

// set cron
func SetCron(c *cron.Cron) {
	wgo.SetCron(c)
}
func (w *WGO) SetCron(c *cron.Cron) {
	if w != nil {
		w.cron = c
	}
}

// add cron
func NewCron(spec string, cmd func()) error {
	return Cron().AddFunc(spec, cmd)
}
