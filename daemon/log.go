package daemon

import (
	"os"
	"wgo/wlog"
)

type (
	logger interface {
		Log(arg0 interface{}, args ...interface{})
	}
)

/* {{{ func (d *Daemon) SetLogger(l logger)
 *
 */
func SetLogger(l logger) {
	if daemon != nil {
		daemon.SetLogger(l)
		return
	}
	Log("[%d][SetLogger] daemon is not registered", os.Getpid())
}
func (d *Daemon) SetLogger(l logger) {
	d.logger = l
}

/* }}} */

/* {{{ func (d *Daemon) Log(arg0 interface{}, args ...interface{})
 *
 */
func Log(arg0 interface{}, args ...interface{}) {
	if daemon != nil {
		daemon.Log(arg0, args...)
		return
	}
	// not set logger
	wlog.Output(arg0, args...)
}
func (d *Daemon) Log(arg0 interface{}, args ...interface{}) {
	if d.logger != nil {
		d.logger.Log(arg0, args...)
		return
	}
	// not set logger
	wlog.Output(arg0, args...)
}

/* }}} */
