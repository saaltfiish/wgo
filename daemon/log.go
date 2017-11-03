package daemon

import (
	"fmt"
	"log"
	"os"
	"strings"
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
	nlog(arg0, args...)
}
func (d *Daemon) Log(arg0 interface{}, args ...interface{}) {
	if d.logger != nil {
		d.logger.Log(arg0, args...)
		return
	}
	// not set logger
	nlog(arg0, args...)
}

/* }}} */

/* {{{ func nlog(arg0 interface{}, args ...interface{})
 * native log
 */
func nlog(arg0 interface{}, args ...interface{}) {
	switch first := arg0.(type) {
	case string:
		// Use the first string as a format string
		log.Printf(first, args...)
	default:
		log.Printf(fmt.Sprint(first)+strings.Repeat(" %v", len(args)), args...)
	}
}

/* }}} */
